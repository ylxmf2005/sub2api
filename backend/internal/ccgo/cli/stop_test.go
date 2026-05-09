package cli

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type stopClientStub struct {
	workspaceID int64
	reason      string
	result      *StopWorkstationResult
}

func (s *stopClientStub) StopWorkstation(_ context.Context, workspaceID int64, reason string) (*StopWorkstationResult, error) {
	s.workspaceID = workspaceID
	s.reason = reason
	return s.result, nil
}

func TestStopUsesExplicitWorkspaceIDAndReason(t *testing.T) {
	client := &stopClientStub{result: &StopWorkstationResult{Workspace: Workspace{ID: 42}, Stopped: true}}
	result, err := Stop(context.Background(), StopOptions{WorkspaceID: 42, Reason: "user stop", Client: client})
	require.NoError(t, err)
	require.Equal(t, int64(42), client.workspaceID)
	require.Equal(t, "user stop", client.reason)
	require.True(t, result.Stopped)
}

func TestStopUsesLastWorkspaceFromConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, SaveConfig(path, Config{
		Server:          "https://ccgo.example.com",
		Token:           "token",
		DeviceID:        "dev_123",
		LastWorkspaceID: 77,
	}))
	client := &stopClientStub{result: &StopWorkstationResult{Workspace: Workspace{ID: 77}, Stopped: true}}

	result, err := Stop(context.Background(), StopOptions{ConfigPath: path, Reason: "ccgo stop", Client: client})
	require.NoError(t, err)
	require.Equal(t, int64(77), client.workspaceID)
	require.Equal(t, "ccgo stop", client.reason)
	require.True(t, result.Stopped)
}
