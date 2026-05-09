package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/hub"
	"github.com/Wei-Shaw/sub2api/internal/ccgo/protocol"
)

const (
	ccgoDeviceLoginTTL             = 10 * time.Minute
	ccgoDeviceLoginPollIntervalSec = 2
)

type CcgoService struct {
	repo          CcgoRepository
	workspaceBase string
	agentHub      *hub.ConnectionManager
	runner        CcgoRunnerController
	userRepo      CcgoDeviceLoginUserReader
	tokenIssuer   CcgoDeviceLoginTokenIssuer
}

type legacyRunnerController struct {
	runStarter CcgoRunStarter
	terminal   CcgoTerminalAttacher
}

func (c legacyRunnerController) StartCcgoRun(ctx context.Context, workspace *CcgoWorkspace) (*CcgoWorkstationRun, bool, error) {
	if c.runStarter == nil {
		return nil, false, ErrCcgoWorkspaceUnavailable.WithMetadata(map[string]string{"component": "runner"})
	}
	return c.runStarter.StartCcgoRun(ctx, workspace)
}

func (c legacyRunnerController) Attach(workspaceID int64, input io.Reader, output io.Writer) error {
	if c.terminal == nil {
		return ErrCcgoWorkspaceUnavailable.WithMetadata(map[string]string{"component": "terminal"})
	}
	return c.terminal.Attach(workspaceID, input, output)
}

func (c legacyRunnerController) RuntimeStatus(context.Context, int64) (*CcgoRunnerStatus, error) {
	return nil, nil
}

func (c legacyRunnerController) StopCcgoRun(context.Context, int64, string) (*CcgoWorkstationRun, *CcgoRunnerStatus, error) {
	return nil, nil, ErrCcgoWorkspaceUnavailable.WithMetadata(map[string]string{"component": "runner"})
}

func NewCcgoService(repo CcgoRepository, agentHub *hub.ConnectionManager, runner CcgoRunnerController, userRepo CcgoDeviceLoginUserReader, tokenIssuer CcgoDeviceLoginTokenIssuer) *CcgoService {
	if agentHub == nil {
		agentHub = hub.NewConnectionManager()
	}
	return &CcgoService{
		repo:          repo,
		workspaceBase: "/var/lib/ccgo/workspaces",
		agentHub:      agentHub,
		runner:        runner,
		userRepo:      userRepo,
		tokenIssuer:   tokenIssuer,
	}
}

func (s *CcgoService) SetRunStarterForTest(starter CcgoRunStarter) {
	if s == nil {
		return
	}
	current := s.runner
	if legacy, ok := current.(legacyRunnerController); ok {
		legacy.runStarter = starter
		s.runner = legacy
		return
	}
	s.runner = legacyRunnerController{runStarter: starter, terminal: current}
}

func (s *CcgoService) SetTerminalAttacherForTest(terminal CcgoTerminalAttacher) {
	if s == nil {
		return
	}
	current := s.runner
	if legacy, ok := current.(legacyRunnerController); ok {
		legacy.terminal = terminal
		s.runner = legacy
		return
	}
	s.runner = legacyRunnerController{runStarter: current, terminal: terminal}
}

func (s *CcgoService) SetRunnerControllerForTest(controller CcgoRunnerController) {
	if s == nil {
		return
	}
	s.runner = controller
}

func (s *CcgoService) SetDeviceLoginDepsForTest(userRepo CcgoDeviceLoginUserReader, tokenIssuer CcgoDeviceLoginTokenIssuer) {
	if s == nil {
		return
	}
	s.userRepo = userRepo
	s.tokenIssuer = tokenIssuer
}

func (s *CcgoService) SetWorkspaceBaseForTest(base string) {
	if s == nil {
		return
	}
	s.workspaceBase = strings.TrimSpace(base)
}

func (s *CcgoService) StartDeviceLogin(ctx context.Context, serverBaseURL, deviceID string) (*CcgoDeviceLoginStartResult, error) {
	if s == nil || s.repo == nil {
		return nil, ErrCcgoWorkspaceUnavailable
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return nil, ErrCcgoInvalidDevice
	}
	deviceCode, err := randomCcgoCode(32)
	if err != nil {
		return nil, err
	}
	userCode, err := randomCcgoUserCode()
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().Add(ccgoDeviceLoginTTL)
	if _, err := s.repo.CreateDeviceLogin(ctx, HashCcgoSecret(deviceCode), HashCcgoSecret(userCode), deviceID, expiresAt); err != nil {
		return nil, err
	}
	return &CcgoDeviceLoginStartResult{
		DeviceCode:      deviceCode,
		UserCode:        userCode,
		VerificationURI: ccgoVerificationURI(serverBaseURL, userCode),
		ExpiresAt:       expiresAt,
		IntervalSeconds: ccgoDeviceLoginPollIntervalSec,
	}, nil
}

func (s *CcgoService) PollDeviceLogin(ctx context.Context, deviceCode string) (*CcgoDeviceLoginPollResult, error) {
	if s == nil || s.repo == nil || s.userRepo == nil || s.tokenIssuer == nil {
		return nil, ErrCcgoWorkspaceUnavailable.WithMetadata(map[string]string{"component": "device_login"})
	}
	login, err := s.repo.FindDeviceLoginByDeviceCodeHash(ctx, HashCcgoSecret(strings.TrimSpace(deviceCode)))
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if login.ExpiresAt.Before(now) {
		if login.Status != CcgoDeviceLoginStatusExpired {
			_, _ = s.repo.ExpireDeviceLogin(ctx, login.ID, now)
		}
		return nil, ErrCcgoDeviceLoginExpired
	}
	switch login.Status {
	case CcgoDeviceLoginStatusPending:
		return nil, ErrCcgoDeviceLoginPending
	case CcgoDeviceLoginStatusConsumed:
		return nil, ErrCcgoDeviceLoginConsumed
	case CcgoDeviceLoginStatusExpired:
		return nil, ErrCcgoDeviceLoginExpired
	case CcgoDeviceLoginStatusApproved:
	default:
		return nil, ErrCcgoDeviceLoginNotFound
	}
	if login.UserID == nil || *login.UserID <= 0 {
		return nil, ErrCcgoDeviceLoginNotFound
	}
	user, err := s.userRepo.GetByID(ctx, *login.UserID)
	if err != nil {
		return nil, err
	}
	if !user.IsActive() {
		return nil, ErrUserNotActive
	}
	s.tokenIssuer.RecordSuccessfulLogin(ctx, user.ID)
	pair, err := s.tokenIssuer.GenerateTokenPair(ctx, user, "")
	if err != nil {
		accessToken, tokenErr := s.tokenIssuer.GenerateToken(user)
		if tokenErr != nil {
			return nil, err
		}
		pair = &TokenPair{AccessToken: accessToken, ExpiresIn: s.tokenIssuer.GetAccessTokenExpiresIn()}
	}
	if _, err := s.repo.ConsumeDeviceLogin(ctx, login.ID, now); err != nil {
		return nil, err
	}
	return &CcgoDeviceLoginPollResult{
		Status:       CcgoDeviceLoginStatusApproved,
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresIn:    pair.ExpiresIn,
		TokenType:    "Bearer",
		UserID:       &user.ID,
		ApprovedAt:   login.ApprovedAt,
	}, nil
}

func (s *CcgoService) ApproveDeviceLogin(ctx context.Context, userID int64, userCode string) (*CcgoDeviceLoginApproveResult, error) {
	if userID <= 0 {
		return nil, ErrCcgoInvalidUser
	}
	if s == nil || s.repo == nil {
		return nil, ErrCcgoWorkspaceUnavailable.WithMetadata(map[string]string{"component": "device_login"})
	}
	login, err := s.repo.FindDeviceLoginByUserCodeHash(ctx, HashCcgoSecret(normalizeCcgoUserCode(userCode)))
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if login.ExpiresAt.Before(now) {
		if login.Status != CcgoDeviceLoginStatusExpired {
			_, _ = s.repo.ExpireDeviceLogin(ctx, login.ID, now)
		}
		return nil, ErrCcgoDeviceLoginExpired
	}
	if login.Status == CcgoDeviceLoginStatusConsumed {
		return nil, ErrCcgoDeviceLoginConsumed
	}
	if login.Status != CcgoDeviceLoginStatusPending && login.Status != CcgoDeviceLoginStatusApproved {
		return nil, ErrCcgoDeviceLoginNotFound
	}
	approved, err := s.repo.ApproveDeviceLogin(ctx, login.ID, userID, now)
	if err != nil {
		return nil, err
	}
	return &CcgoDeviceLoginApproveResult{
		Status:    approved.Status,
		UserID:    userID,
		ExpiresAt: approved.ExpiresAt,
	}, nil
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
	if s == nil || s.agentHub == nil {
		return nil, ErrCcgoWorkspaceUnavailable
	}
	workspace, err := s.authorizeWorkspace(ctx, input.UserID, input.WorkspaceID)
	if err != nil {
		return nil, err
	}
	agent, err := s.AgentConnectionStatus(workspace.ID)
	if err != nil {
		if isProtocolErrorCode(err, protocol.ErrorAgentDisconnected) {
			return nil, ErrCcgoAgentDisconnected.WithCause(err)
		}
		return nil, err
	}
	if s.runner != nil {
		run, reused, err := s.runner.StartCcgoRun(ctx, workspace)
		if err != nil {
			return nil, err
		}
		return &CcgoStartWorkstationResult{Workspace: workspace, Run: run, Agent: agent, Reused: reused}, nil
	}
	return nil, ErrCcgoWorkspaceUnavailable.WithMetadata(map[string]string{"component": "runner"})
}

func (s *CcgoService) WorkstationStatus(ctx context.Context, input CcgoWorkstationStatusInput) (*CcgoWorkstationStatusResult, error) {
	workspace, err := s.authorizeWorkspace(ctx, input.UserID, input.WorkspaceID)
	if err != nil {
		return nil, err
	}
	var agent *CcgoAgentConnection
	agentConnected := false
	agentError := ""
	if status, err := s.AgentConnectionStatus(workspace.ID); err != nil {
		if isProtocolErrorCode(err, protocol.ErrorAgentDisconnected) {
			agentError = ErrCcgoAgentDisconnected.Message
		} else {
			return nil, err
		}
	} else {
		agent = status
		agentConnected = true
	}
	var runnerStatus *CcgoRunnerStatus
	if s.runner != nil {
		runnerStatus, err = s.runner.RuntimeStatus(ctx, workspace.ID)
		if err != nil {
			return nil, err
		}
	}
	if runnerStatus == nil {
		runnerStatus = &CcgoRunnerStatus{WorkspaceID: workspace.ID, LastCheckedAt: time.Now()}
	}
	latestRun, err := s.repo.FindLatestWorkstationRun(ctx, workspace.ID)
	if err != nil {
		return nil, err
	}
	return &CcgoWorkstationStatusResult{
		Workspace:      workspace,
		Agent:          agent,
		AgentConnected: agentConnected,
		AgentError:     agentError,
		Runner:         runnerStatus,
		LatestRun:      latestRun,
		Resumable:      latestRun == nil || !runnerStatus.Running,
	}, nil
}

func (s *CcgoService) StopWorkstation(ctx context.Context, input CcgoStopWorkstationInput) (*CcgoStopWorkstationResult, error) {
	workspace, err := s.authorizeWorkspace(ctx, input.UserID, input.WorkspaceID)
	if err != nil {
		return nil, err
	}
	if s == nil || s.runner == nil {
		return nil, ErrCcgoWorkspaceUnavailable.WithMetadata(map[string]string{"component": "runner"})
	}
	run, runnerStatus, err := s.runner.StopCcgoRun(ctx, workspace.ID, input.Reason)
	if err != nil {
		return nil, err
	}
	if runnerStatus == nil {
		runnerStatus = &CcgoRunnerStatus{WorkspaceID: workspace.ID, LastCheckedAt: time.Now()}
	}
	agentDisconnected := false
	if s.agentHub != nil {
		s.agentHub.Unregister(workspace.ID)
		agentDisconnected = true
	}
	return &CcgoStopWorkstationResult{
		Workspace:         workspace,
		Run:               run,
		Runner:            runnerStatus,
		Stopped:           run != nil || !runnerStatus.Running,
		AgentDisconnected: agentDisconnected,
	}, nil
}

func (s *CcgoService) AttachTerminal(ctx context.Context, userID int64, workspaceID int64, input io.Reader, output io.Writer) error {
	if s == nil || s.runner == nil {
		return ErrCcgoWorkspaceUnavailable.WithMetadata(map[string]string{"component": "terminal"})
	}
	workspace, err := s.authorizeWorkspace(ctx, userID, workspaceID)
	if err != nil {
		return err
	}
	return s.runner.Attach(workspace.ID, input, output)
}

func (s *CcgoService) authorizeWorkspace(ctx context.Context, userID int64, workspaceID int64) (*CcgoWorkspace, error) {
	if userID <= 0 {
		return nil, ErrCcgoInvalidUser
	}
	if workspaceID <= 0 {
		return nil, protocol.NewError(protocol.ErrorInvalidRequest, "workspace id is required")
	}
	if s == nil || s.repo == nil {
		return nil, ErrCcgoWorkspaceUnavailable
	}
	workspace, err := s.repo.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	if workspace.UserID != userID {
		return nil, ErrCcgoWorkspaceForbidden
	}
	return workspace, nil
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

func randomCcgoCode(bytesLen int) (string, error) {
	buf := make([]byte, bytesLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func randomCcgoUserCode() (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, len(buf)+1)
	for i, b := range buf {
		if i == 4 {
			out[i] = '-'
		}
		target := i
		if i >= 4 {
			target = i + 1
		}
		out[target] = alphabet[int(b)%len(alphabet)]
	}
	return string(out), nil
}

func normalizeCcgoUserCode(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "")
	if len(value) == 8 && !strings.Contains(value, "-") {
		return value[:4] + "-" + value[4:]
	}
	return value
}

func ccgoVerificationURI(serverBaseURL, userCode string) string {
	base := strings.TrimRight(strings.TrimSpace(serverBaseURL), "/")
	if base == "" {
		base = "/"
	}
	parsed, err := url.Parse(base)
	if err != nil {
		return "/ccgo/device?code=" + url.QueryEscape(userCode)
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/ccgo/device"
	query := parsed.Query()
	query.Set("code", userCode)
	parsed.RawQuery = query.Encode()
	return parsed.String()
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
