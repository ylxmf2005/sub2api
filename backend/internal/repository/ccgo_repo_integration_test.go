//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestCcgoRepository_ResolveWorkspace_CreatesAndReusesStableMapping(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	txCtx := dbent.NewTxContext(ctx, tx)
	client := tx.Client()
	repo := NewCcgoRepository(client)
	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("ccgo-workspace-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
		Concurrency:  5,
	})
	input := service.CcgoResolveWorkspaceInput{
		UserID:           user.ID,
		CanonicalRoot:    "/Users/alice/project",
		LocalRootDisplay: "/Users/alice/project",
		LocalRootHash:    service.HashCcgoSecret("/Users/alice/project"),
		OS:               service.CcgoOSDarwin,
		PathStyle:        service.CcgoPathStylePOSIX,
		DeviceID:         "device-a",
		WorkspaceBase:    "/tmp/ccgo-workspaces",
	}

	first, err := repo.ResolveWorkspace(txCtx, input)
	require.NoError(t, err)
	require.True(t, first.Created)
	require.NotNil(t, first.Workspace)
	require.Contains(t, first.Workspace.ServerRoot, "/tmp/ccgo-workspaces")
	require.Equal(t, ".../project", first.Workspace.LocalRootRedacted)

	second, err := repo.ResolveWorkspace(txCtx, input)
	require.NoError(t, err)
	require.False(t, second.Created)
	require.Equal(t, first.Workspace.ID, second.Workspace.ID)
	require.Equal(t, first.Workspace.ServerRoot, second.Workspace.ServerRoot)
}

func TestCcgoRepository_ResolveWorkspace_RejectsSamePathDifferentDevice(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	txCtx := dbent.NewTxContext(ctx, tx)
	client := tx.Client()
	repo := NewCcgoRepository(client)
	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("ccgo-device-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
		Concurrency:  5,
	})
	input := service.CcgoResolveWorkspaceInput{
		UserID:           user.ID,
		CanonicalRoot:    "/home/alice/project",
		LocalRootDisplay: "/home/alice/project",
		LocalRootHash:    service.HashCcgoSecret("/home/alice/project"),
		OS:               service.CcgoOSLinux,
		PathStyle:        service.CcgoPathStylePOSIX,
		DeviceID:         "device-a",
		WorkspaceBase:    "/tmp/ccgo-workspaces",
	}
	_, err := repo.ResolveWorkspace(txCtx, input)
	require.NoError(t, err)

	input.DeviceID = "device-b"
	_, err = repo.ResolveWorkspace(txCtx, input)
	require.Error(t, err)
	require.True(t, errors.IsConflict(err))
	require.Equal(t, "CCGO_DEVICE_MISMATCH", errors.Reason(err))
}

func TestCcgoService_ValidateAgentCredential_AllowsOnlyOneUse(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	txCtx := dbent.NewTxContext(ctx, tx)
	client := tx.Client()
	repo := NewCcgoRepository(client)
	ccgoService := service.NewCcgoService(repo, nil, nil, nil, nil)
	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("ccgo-credential-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
		Concurrency:  5,
	})
	resolution, err := ccgoService.ResolveWorkspace(txCtx, service.CcgoResolveWorkspaceInput{
		UserID:           user.ID,
		CanonicalRoot:    "/Users/alice/project",
		LocalRootDisplay: "/Users/alice/project",
		LocalRootHash:    service.HashCcgoSecret("/Users/alice/project"),
		OS:               service.CcgoOSDarwin,
		PathStyle:        service.CcgoPathStylePOSIX,
		DeviceID:         "device-a",
		WorkspaceBase:    "/tmp/ccgo-workspaces",
	})
	require.NoError(t, err)
	require.NotEmpty(t, resolution.AgentCredential.Token)
	require.NotEmpty(t, resolution.AgentCredential.Nonce)

	credential, err := ccgoService.ValidateAgentCredential(txCtx, resolution.AgentCredential.Token, resolution.AgentCredential.Nonce)
	require.NoError(t, err)
	require.Equal(t, resolution.Workspace.ID, credential.WorkspaceID)

	_, err = ccgoService.ValidateAgentCredential(txCtx, resolution.AgentCredential.Token, resolution.AgentCredential.Nonce)
	require.ErrorIs(t, err, service.ErrCcgoAgentCredentialReplayed)
}
