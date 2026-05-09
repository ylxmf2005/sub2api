package cli

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type statusClientStub struct {
	workspaceID int64
	result      *WorkstationStatusResult
}

func (s *statusClientStub) WorkstationStatus(_ context.Context, workspaceID int64) (*WorkstationStatusResult, error) {
	s.workspaceID = workspaceID
	return s.result, nil
}

func TestStatusUsesExplicitWorkspaceID(t *testing.T) {
	client := &statusClientStub{result: &WorkstationStatusResult{Workspace: Workspace{ID: 42}}}
	result, err := Status(context.Background(), StatusOptions{WorkspaceID: 42, Client: client})
	require.NoError(t, err)
	require.Equal(t, int64(42), client.workspaceID)
	require.Equal(t, int64(42), result.Workspace.ID)
}

func TestStatusUsesLastWorkspaceFromConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, SaveConfig(path, Config{
		Server:          "https://ccgo.example.com",
		Token:           "token",
		DeviceID:        "dev_123",
		LastWorkspaceID: 77,
	}))
	client := &statusClientStub{result: &WorkstationStatusResult{Workspace: Workspace{ID: 77}}}

	result, err := Status(context.Background(), StatusOptions{ConfigPath: path, Client: client})
	require.NoError(t, err)
	require.Equal(t, int64(77), client.workspaceID)
	require.Equal(t, int64(77), result.Workspace.ID)
}

func TestStatusRequiresWorkspaceWhenNoLastWorkspace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, SaveConfig(path, Config{
		Server:   "https://ccgo.example.com",
		Token:    "token",
		DeviceID: "dev_123",
	}))

	_, err := Status(context.Background(), StatusOptions{ConfigPath: path, Client: &statusClientStub{}})
	require.ErrorContains(t, err, "workspace id")
}
