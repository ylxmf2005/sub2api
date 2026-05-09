package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/hub"
	"github.com/Wei-Shaw/sub2api/internal/ccgo/protocol"
)

type CcgoService struct {
	repo          CcgoRepository
	workspaceBase string
	agentHub      *hub.ConnectionManager
	runStarter    CcgoRunStarter
	terminal      CcgoTerminalAttacher
}

func NewCcgoService(repo CcgoRepository, agentHub *hub.ConnectionManager, runStarter CcgoRunStarter, terminal CcgoTerminalAttacher) *CcgoService {
	if agentHub == nil {
		agentHub = hub.NewConnectionManager()
	}
	return &CcgoService{repo: repo, workspaceBase: "/var/lib/ccgo/workspaces", agentHub: agentHub, runStarter: runStarter, terminal: terminal}
}

func (s *CcgoService) SetRunStarterForTest(starter CcgoRunStarter) {
	if s == nil {
		return
	}
	s.runStarter = starter
}

func (s *CcgoService) SetTerminalAttacherForTest(terminal CcgoTerminalAttacher) {
	if s == nil {
		return
	}
	s.terminal = terminal
}

func (s *CcgoService) SetWorkspaceBaseForTest(base string) {
	if s == nil {
		return
	}
	s.workspaceBase = strings.TrimSpace(base)
}

func (s *CcgoService) ResolveWorkspace(ctx context.Context, input CcgoResolveWorkspaceInput) (*CcgoWorkspaceResolution, error) {
	if err := validateCcgoResolveWorkspaceInput(input); err != nil {
		return nil, err
	}
	if s == nil || s.repo == nil {
		return nil, ErrCcgoWorkspaceUnavailable
	}
	if strings.TrimSpace(input.WorkspaceBase) == "" {
		input.WorkspaceBase = s.workspaceBase
	}
	resolution, err := s.repo.ResolveWorkspace(ctx, input)
	if err != nil {
		return nil, err
	}
	credential, err := s.repo.IssueAgentCredential(ctx, resolution.Workspace, DefaultCcgoAgentCredentialTTL)
	if err != nil {
		return nil, err
	}
	resolution.AgentCredential = credential
	return resolution, nil
}

func (s *CcgoService) ValidateAgentCredential(ctx context.Context, token, nonce string) (*CcgoAgentCredential, error) {
	token = strings.TrimSpace(token)
	nonce = strings.TrimSpace(nonce)
	if token == "" || nonce == "" {
		return nil, ErrCcgoAgentCredentialInvalid
	}
	if s == nil || s.repo == nil {
		return nil, ErrCcgoWorkspaceUnavailable
	}
	credential, err := s.repo.FindAgentCredentialByTokenHash(ctx, HashCcgoSecret(token))
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if credential.ExpiresAt.Before(now) {
		return nil, ErrCcgoAgentCredentialExpired
	}
	if credential.RevokedAt != nil || credential.Status == CcgoAgentCredentialStatusRevoked {
		return nil, ErrCcgoAgentCredentialRevoked
	}
	if credential.UsedAt != nil || credential.Status == CcgoAgentCredentialStatusUsed {
		return nil, ErrCcgoAgentCredentialReplayed
	}
	if credential.NonceHash != HashCcgoSecret(nonce) {
		return nil, ErrCcgoAgentCredentialInvalid
	}
	if err := s.repo.MarkAgentCredentialUsed(ctx, credential.ID); err != nil {
		return nil, err
	}
	return credential, nil
}

func (s *CcgoService) RegisterAgentConnection(ctx context.Context, token, nonce string, transport hub.Transport) (*CcgoAgentConnection, error) {
	if s == nil || s.agentHub == nil || transport == nil {
		return nil, ErrCcgoWorkspaceUnavailable
	}
	credential, err := s.ValidateAgentCredential(ctx, token, nonce)
	if err != nil {
		return nil, err
	}
	conn := s.agentHub.Register(credential.WorkspaceID, transport)
	return &CcgoAgentConnection{
		WorkspaceID: credential.WorkspaceID,
		UserID:      credential.UserID,
		LastSeenAt:  conn.LastSeen(),
	}, nil
}

func (s *CcgoService) AgentConnectionStatus(workspaceID int64) (*CcgoAgentConnection, error) {
	if s == nil || s.agentHub == nil {
		return nil, ErrCcgoWorkspaceUnavailable
	}
	conn, ok := s.agentHub.Get(workspaceID)
	if !ok {
		return nil, protocol.NewError(protocol.ErrorAgentDisconnected, "agent is not connected")
	}
	return &CcgoAgentConnection{WorkspaceID: workspaceID, LastSeenAt: conn.LastSeen()}, nil
}

func (s *CcgoService) FileStat(ctx context.Context, workspaceID int64, req protocol.FileStatRequest) (protocol.FileStatResponse, error) {
	if s == nil || s.agentHub == nil {
		return protocol.FileStatResponse{}, ErrCcgoWorkspaceUnavailable
	}
	if workspaceID <= 0 {
		return protocol.FileStatResponse{}, protocol.NewError(protocol.ErrorInvalidRequest, "workspace id is required")
	}
	return s.agentHub.FileStat(ctx, workspaceID, req)
}

func (s *CcgoService) Exec(ctx context.Context, workspaceID int64, req protocol.ExecRequest) (protocol.ExecResponse, error) {
	if s == nil || s.agentHub == nil {
		return protocol.ExecResponse{}, ErrCcgoWorkspaceUnavailable
	}
	if workspaceID <= 0 {
		return protocol.ExecResponse{}, protocol.NewError(protocol.ErrorInvalidRequest, "workspace id is required")
	}
	if strings.TrimSpace(req.Command) == "" {
		return protocol.ExecResponse{}, protocol.NewError(protocol.ErrorInvalidRequest, "command is required")
	}
	return s.agentHub.Exec(ctx, workspaceID, req)
}

func (s *CcgoService) StartWorkstation(ctx context.Context, input CcgoStartWorkstationInput) (*CcgoStartWorkstationResult, error) {
	if input.UserID <= 0 {
		return nil, ErrCcgoInvalidUser
	}
	if input.WorkspaceID <= 0 {
		return nil, protocol.NewError(protocol.ErrorInvalidRequest, "workspace id is required")
	}
	if s == nil || s.repo == nil || s.agentHub == nil {
		return nil, ErrCcgoWorkspaceUnavailable
	}
	workspace, err := s.repo.GetWorkspace(ctx, input.WorkspaceID)
	if err != nil {
		return nil, err
	}
	if workspace.UserID != input.UserID {
		return nil, ErrCcgoWorkspaceForbidden
	}
	agent, err := s.AgentConnectionStatus(workspace.ID)
	if err != nil {
		if isProtocolErrorCode(err, protocol.ErrorAgentDisconnected) {
			return nil, ErrCcgoAgentDisconnected.WithCause(err)
		}
		return nil, err
	}
	if s.runStarter != nil {
		run, reused, err := s.runStarter.StartCcgoRun(ctx, workspace)
		if err != nil {
			return nil, err
		}
		return &CcgoStartWorkstationResult{Workspace: workspace, Run: run, Agent: agent, Reused: reused}, nil
	}
	return nil, ErrCcgoWorkspaceUnavailable.WithMetadata(map[string]string{"component": "runner"})
}

func (s *CcgoService) AttachTerminal(ctx context.Context, userID int64, workspaceID int64, input io.Reader, output io.Writer) error {
	if userID <= 0 {
		return ErrCcgoInvalidUser
	}
	if workspaceID <= 0 {
		return protocol.NewError(protocol.ErrorInvalidRequest, "workspace id is required")
	}
	if s == nil || s.repo == nil || s.terminal == nil {
		return ErrCcgoWorkspaceUnavailable.WithMetadata(map[string]string{"component": "terminal"})
	}
	workspace, err := s.repo.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return err
	}
	if workspace.UserID != userID {
		return ErrCcgoWorkspaceForbidden
	}
	return s.terminal.Attach(workspaceID, input, output)
}

func isProtocolErrorCode(err error, code string) bool {
	var protocolErr *protocol.Error
	return errors.As(err, &protocolErr) && protocolErr.Code == code
}

func ParseCcgoWorkspaceID(value string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || id <= 0 {
		return 0, protocol.NewError(protocol.ErrorInvalidRequest, "workspace id is required")
	}
	return id, nil
}

func HashCcgoSecret(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func validateCcgoResolveWorkspaceInput(input CcgoResolveWorkspaceInput) error {
	if input.UserID <= 0 {
		return ErrCcgoInvalidUser
	}
	if strings.TrimSpace(input.CanonicalRoot) == "" || strings.TrimSpace(input.LocalRootDisplay) == "" {
		return ErrCcgoInvalidLocalRoot
	}
	if len(strings.TrimSpace(input.LocalRootHash)) != 64 {
		return ErrCcgoInvalidLocalRootHash
	}
	if strings.TrimSpace(input.DeviceID) == "" {
		return ErrCcgoInvalidDevice
	}
	switch input.OS {
	case CcgoOSDarwin, CcgoOSLinux, CcgoOSWindows:
	default:
		return ErrCcgoInvalidOS
	}
	switch input.PathStyle {
	case CcgoPathStylePOSIX, CcgoPathStyleWindows:
	default:
		return ErrCcgoInvalidPathStyle
	}
	return nil
}
