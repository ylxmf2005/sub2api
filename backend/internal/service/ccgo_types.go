package service

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	CcgoOSDarwin  = "darwin"
	CcgoOSLinux   = "linux"
	CcgoOSWindows = "windows"

	CcgoPathStylePOSIX   = "posix"
	CcgoPathStyleWindows = "windows"

	CcgoWorkspaceStatusActive   = "active"
	CcgoWorkspaceStatusArchived = "archived"

	CcgoAgentCredentialStatusActive  = "active"
	CcgoAgentCredentialStatusUsed    = "used"
	CcgoAgentCredentialStatusRevoked = "revoked"
	CcgoAgentCredentialStatusExpired = "expired"

	DefaultCcgoAgentCredentialTTL = 15 * time.Minute
)

var (
	ErrCcgoInvalidUser             = infraerrors.BadRequest("CCGO_INVALID_USER", "invalid ccgo user")
	ErrCcgoInvalidLocalRoot        = infraerrors.BadRequest("CCGO_INVALID_LOCAL_ROOT", "invalid ccgo local root")
	ErrCcgoInvalidLocalRootHash    = infraerrors.BadRequest("CCGO_INVALID_LOCAL_ROOT_HASH", "invalid ccgo local root hash")
	ErrCcgoInvalidDevice           = infraerrors.BadRequest("CCGO_INVALID_DEVICE", "invalid ccgo device")
	ErrCcgoInvalidOS               = infraerrors.BadRequest("CCGO_INVALID_OS", "invalid ccgo operating system")
	ErrCcgoInvalidPathStyle        = infraerrors.BadRequest("CCGO_INVALID_PATH_STYLE", "invalid ccgo path style")
	ErrCcgoDeviceMismatch          = infraerrors.Conflict("CCGO_DEVICE_MISMATCH", "ccgo workspace is already bound to another device")
	ErrCcgoWorkspaceUnavailable    = infraerrors.ServiceUnavailable("CCGO_WORKSPACE_UNAVAILABLE", "ccgo workspace service unavailable")
	ErrCcgoAgentCredentialExpired  = infraerrors.Unauthorized("CCGO_AGENT_CREDENTIAL_EXPIRED", "ccgo agent credential has expired")
	ErrCcgoAgentCredentialRevoked  = infraerrors.Unauthorized("CCGO_AGENT_CREDENTIAL_REVOKED", "ccgo agent credential has been revoked")
	ErrCcgoAgentCredentialInvalid  = infraerrors.Unauthorized("CCGO_AGENT_CREDENTIAL_INVALID", "invalid ccgo agent credential")
	ErrCcgoAgentCredentialReplayed = infraerrors.Unauthorized("CCGO_AGENT_CREDENTIAL_REPLAYED", "ccgo agent credential has already been used")
)

type CcgoWorkspace struct {
	ID                int64      `json:"id"`
	UserID            int64      `json:"user_id"`
	WorkspaceSlug     string     `json:"workspace_slug"`
	ServerRoot        string     `json:"server_root"`
	LocalRootHash     string     `json:"local_root_hash"`
	LocalRootDisplay  string     `json:"local_root_display"`
	LocalRootRedacted string     `json:"local_root_redacted"`
	OS                string     `json:"os"`
	PathStyle         string     `json:"path_style"`
	DeviceID          string     `json:"device_id"`
	Status            string     `json:"status"`
	LastSeenAt        *time.Time `json:"last_seen_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type CcgoResolveWorkspaceInput struct {
	UserID           int64
	CanonicalRoot    string
	LocalRootHash    string
	LocalRootDisplay string
	OS               string
	PathStyle        string
	DeviceID         string
	WorkspaceBase    string
}

type CcgoWorkspaceResolution struct {
	Workspace       *CcgoWorkspace          `json:"workspace"`
	AgentCredential *CcgoIssuedCredential   `json:"agent_credential"`
	Created         bool                    `json:"created"`
	DeviceConflict  *CcgoDeviceConflictInfo `json:"device_conflict,omitempty"`
}

type CcgoIssuedCredential struct {
	Token     string    `json:"token"`
	Nonce     string    `json:"nonce"`
	ExpiresAt time.Time `json:"expires_at"`
}

type CcgoDeviceConflictInfo struct {
	ExistingDeviceID  string `json:"existing_device_id"`
	RequestedDeviceID string `json:"requested_device_id"`
}

type CcgoAgentCredential struct {
	ID          int64
	WorkspaceID int64
	UserID      int64
	TokenHash   string
	NonceHash   string
	Status      string
	ExpiresAt   time.Time
	UsedAt      *time.Time
	RevokedAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CcgoRepository interface {
	ResolveWorkspace(ctx context.Context, input CcgoResolveWorkspaceInput) (*CcgoWorkspaceResolution, error)
	IssueAgentCredential(ctx context.Context, workspace *CcgoWorkspace, ttl time.Duration) (*CcgoIssuedCredential, error)
	FindAgentCredentialByTokenHash(ctx context.Context, tokenHash string) (*CcgoAgentCredential, error)
	MarkAgentCredentialUsed(ctx context.Context, credentialID int64) error
}
