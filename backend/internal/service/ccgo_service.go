package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

type CcgoService struct {
	repo          CcgoRepository
	workspaceBase string
}

func NewCcgoService(repo CcgoRepository) *CcgoService {
	return &CcgoService{repo: repo, workspaceBase: "/var/lib/ccgo/workspaces"}
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
