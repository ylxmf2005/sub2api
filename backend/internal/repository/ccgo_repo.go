package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/ccgoagentcredential"
	"github.com/Wei-Shaw/sub2api/ent/ccgoworkspace"
	"github.com/Wei-Shaw/sub2api/ent/ccgoworkstationrun"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

const (
	ccgoCredentialBytes = 32
	ccgoNonceBytes      = 16
)

type ccgoRepository struct {
	client *dbent.Client
}

func NewCcgoRepository(client *dbent.Client) service.CcgoRepository {
	return &ccgoRepository{client: client}
}

func (r *ccgoRepository) ResolveWorkspace(ctx context.Context, input service.CcgoResolveWorkspaceInput) (*service.CcgoWorkspaceResolution, error) {
	client := clientFromContext(ctx, r.client)
	now := time.Now()
	existing, err := client.CcgoWorkspace.Query().
		Where(
			ccgoworkspace.UserIDEQ(input.UserID),
			ccgoworkspace.LocalRootHashEQ(input.LocalRootHash),
		).
		Only(ctx)
	if err == nil {
		workspace := ccgoWorkspaceFromEnt(existing)
		if existing.DeviceID != "" && input.DeviceID != "" && existing.DeviceID != input.DeviceID {
			return nil, service.ErrCcgoDeviceMismatch.WithMetadata(map[string]string{
				"existing_device_id":  existing.DeviceID,
				"requested_device_id": input.DeviceID,
			})
		}
		if _, err := client.CcgoWorkspace.UpdateOneID(existing.ID).
			SetLastSeenAt(now).
			Save(ctx); err != nil {
			return nil, err
		}
		workspace.LastSeenAt = &now
		return &service.CcgoWorkspaceResolution{Workspace: workspace}, nil
	}
	if !dbent.IsNotFound(err) {
		return nil, err
	}

	workspaceSlug := ccgoWorkspaceSlug(input.UserID, input.LocalRootHash)
	serverRoot := ccgoServerRoot(input.WorkspaceBase, input.UserID, workspaceSlug)
	created, err := client.CcgoWorkspace.Create().
		SetUserID(input.UserID).
		SetWorkspaceSlug(workspaceSlug).
		SetServerRoot(serverRoot).
		SetLocalRootHash(input.LocalRootHash).
		SetLocalRootDisplay(strings.TrimSpace(input.LocalRootDisplay)).
		SetLocalRootRedacted(redactCcgoPath(input.LocalRootDisplay)).
		SetOs(input.OS).
		SetPathStyle(input.PathStyle).
		SetDeviceID(strings.TrimSpace(input.DeviceID)).
		SetStatus(service.CcgoWorkspaceStatusActive).
		SetLastSeenAt(now).
		Save(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, nil, service.ErrCcgoDeviceMismatch)
	}
	workspace := ccgoWorkspaceFromEnt(created)
	return &service.CcgoWorkspaceResolution{Workspace: workspace, Created: true}, nil
}

func (r *ccgoRepository) IssueAgentCredential(ctx context.Context, workspace *service.CcgoWorkspace, ttl time.Duration) (*service.CcgoIssuedCredential, error) {
	if workspace == nil || workspace.ID <= 0 || workspace.UserID <= 0 {
		return nil, service.ErrCcgoWorkspaceUnavailable
	}
	if ttl <= 0 {
		ttl = service.DefaultCcgoAgentCredentialTTL
	}
	token, err := randomHex(ccgoCredentialBytes)
	if err != nil {
		return nil, err
	}
	nonce, err := randomHex(ccgoNonceBytes)
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().Add(ttl)
	if _, err := clientFromContext(ctx, r.client).CcgoAgentCredential.Create().
		SetWorkspaceID(workspace.ID).
		SetUserID(workspace.UserID).
		SetTokenHash(service.HashCcgoSecret(token)).
		SetNonceHash(service.HashCcgoSecret(nonce)).
		SetExpiresAt(expiresAt).
		SetStatus(service.CcgoAgentCredentialStatusActive).
		Save(ctx); err != nil {
		return nil, err
	}
	return &service.CcgoIssuedCredential{Token: token, Nonce: nonce, ExpiresAt: expiresAt}, nil
}

func (r *ccgoRepository) FindAgentCredentialByTokenHash(ctx context.Context, tokenHash string) (*service.CcgoAgentCredential, error) {
	credential, err := clientFromContext(ctx, r.client).CcgoAgentCredential.Query().
		Where(ccgoagentcredential.TokenHashEQ(tokenHash)).
		Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrCcgoAgentCredentialInvalid, nil)
	}
	return ccgoAgentCredentialFromEnt(credential), nil
}

func (r *ccgoRepository) MarkAgentCredentialUsed(ctx context.Context, credentialID int64) error {
	if credentialID <= 0 {
		return service.ErrCcgoAgentCredentialInvalid
	}
	now := time.Now()
	return clientFromContext(ctx, r.client).CcgoAgentCredential.UpdateOneID(credentialID).
		SetStatus(service.CcgoAgentCredentialStatusUsed).
		SetUsedAt(now).
		Exec(ctx)
}

func (r *ccgoRepository) GetWorkspace(ctx context.Context, workspaceID int64) (*service.CcgoWorkspace, error) {
	if workspaceID <= 0 {
		return nil, service.ErrCcgoWorkspaceNotFound
	}
	workspace, err := clientFromContext(ctx, r.client).CcgoWorkspace.Get(ctx, workspaceID)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrCcgoWorkspaceNotFound, nil)
	}
	return ccgoWorkspaceFromEnt(workspace), nil
}

func (r *ccgoRepository) FindActiveWorkstationRun(ctx context.Context, workspaceID int64) (*service.CcgoWorkstationRun, error) {
	if workspaceID <= 0 {
		return nil, service.ErrCcgoWorkspaceNotFound
	}
	run, err := clientFromContext(ctx, r.client).CcgoWorkstationRun.Query().
		Where(
			ccgoworkstationrun.WorkspaceIDEQ(workspaceID),
			ccgoworkstationrun.StatusIn(service.CcgoRunStatusStarting, service.CcgoRunStatusRunning),
		).
		Order(dbent.Desc(ccgoworkstationrun.FieldCreatedAt)).
		First(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, nil
		}
		return nil, translatePersistenceError(err, service.ErrCcgoWorkspaceNotFound, nil)
	}
	return ccgoWorkstationRunFromEnt(run), nil
}

func (r *ccgoRepository) CreateWorkstationRun(ctx context.Context, workspace *service.CcgoWorkspace, runID string, now time.Time) (*service.CcgoWorkstationRun, error) {
	if workspace == nil || workspace.ID <= 0 || workspace.UserID <= 0 {
		return nil, service.ErrCcgoWorkspaceNotFound
	}
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return nil, service.ErrCcgoWorkspaceUnavailable
	}
	run, err := clientFromContext(ctx, r.client).CcgoWorkstationRun.Create().
		SetWorkspaceID(workspace.ID).
		SetUserID(workspace.UserID).
		SetRunID(runID).
		SetStatus(service.CcgoRunStatusStarting).
		SetStartedAt(now).
		SetLastHeartbeatAt(now).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return ccgoWorkstationRunFromEnt(run), nil
}

func (r *ccgoRepository) MarkWorkstationRunRunning(ctx context.Context, runID string, serverPID string, now time.Time) (*service.CcgoWorkstationRun, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return nil, service.ErrCcgoWorkspaceUnavailable
	}
	run, err := clientFromContext(ctx, r.client).CcgoWorkstationRun.Query().
		Where(ccgoworkstationrun.RunIDEQ(runID)).
		Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrCcgoWorkspaceNotFound, nil)
	}
	updated, err := clientFromContext(ctx, r.client).CcgoWorkstationRun.UpdateOne(run).
		SetStatus(service.CcgoRunStatusRunning).
		SetServerPid(strings.TrimSpace(serverPID)).
		SetLastHeartbeatAt(now).
		Save(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrCcgoWorkspaceNotFound, nil)
	}
	return ccgoWorkstationRunFromEnt(updated), nil
}

func (r *ccgoRepository) MarkWorkstationRunFailed(ctx context.Context, runID string, reason string, now time.Time) error {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return service.ErrCcgoWorkspaceUnavailable
	}
	run, err := clientFromContext(ctx, r.client).CcgoWorkstationRun.Query().
		Where(ccgoworkstationrun.RunIDEQ(runID)).
		Only(ctx)
	if err != nil {
		return translatePersistenceError(err, service.ErrCcgoWorkspaceNotFound, nil)
	}
	return clientFromContext(ctx, r.client).CcgoWorkstationRun.UpdateOne(run).
		SetStatus(service.CcgoRunStatusFailed).
		SetStoppedAt(now).
		SetStopReason(strings.TrimSpace(reason)).
		Exec(ctx)
}

func ccgoWorkspaceFromEnt(w *dbent.CcgoWorkspace) *service.CcgoWorkspace {
	if w == nil {
		return nil
	}
	return &service.CcgoWorkspace{
		ID:                w.ID,
		UserID:            w.UserID,
		WorkspaceSlug:     w.WorkspaceSlug,
		ServerRoot:        w.ServerRoot,
		LocalRootHash:     w.LocalRootHash,
		LocalRootDisplay:  w.LocalRootDisplay,
		LocalRootRedacted: w.LocalRootRedacted,
		OS:                w.Os,
		PathStyle:         w.PathStyle,
		DeviceID:          w.DeviceID,
		Status:            w.Status,
		LastSeenAt:        w.LastSeenAt,
		CreatedAt:         w.CreatedAt,
		UpdatedAt:         w.UpdatedAt,
	}
}

func ccgoWorkstationRunFromEnt(r *dbent.CcgoWorkstationRun) *service.CcgoWorkstationRun {
	if r == nil {
		return nil
	}
	return &service.CcgoWorkstationRun{
		ID:              r.ID,
		WorkspaceID:     r.WorkspaceID,
		UserID:          r.UserID,
		RunID:           r.RunID,
		Status:          r.Status,
		ServerPID:       r.ServerPid,
		StartedAt:       r.StartedAt,
		StoppedAt:       r.StoppedAt,
		StopReason:      r.StopReason,
		LastHeartbeatAt: r.LastHeartbeatAt,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
	}
}

func ccgoAgentCredentialFromEnt(c *dbent.CcgoAgentCredential) *service.CcgoAgentCredential {
	if c == nil {
		return nil
	}
	return &service.CcgoAgentCredential{
		ID:          c.ID,
		WorkspaceID: c.WorkspaceID,
		UserID:      c.UserID,
		TokenHash:   c.TokenHash,
		NonceHash:   c.NonceHash,
		Status:      c.Status,
		ExpiresAt:   c.ExpiresAt,
		UsedAt:      c.UsedAt,
		RevokedAt:   c.RevokedAt,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}

func ccgoWorkspaceSlug(userID int64, localRootHash string) string {
	hash := strings.TrimSpace(localRootHash)
	if len(hash) > 16 {
		hash = hash[:16]
	}
	return fmt.Sprintf("u%d-%s", userID, hash)
}

func ccgoServerRoot(base string, userID int64, workspaceSlug string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "/var/lib/ccgo/workspaces"
	}
	return path.Join(base, fmt.Sprintf("user-%d", userID), workspaceSlug)
}

func redactCcgoPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	clean := strings.TrimRight(strings.ReplaceAll(value, "\\", "/"), "/")
	if clean == "" {
		return value
	}
	parts := strings.Split(clean, "/")
	last := parts[len(parts)-1]
	if last == "" {
		return "***"
	}
	return path.Join("...", last)
}

func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate ccgo secret: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
