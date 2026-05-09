package runner

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/protocol"
	"github.com/Wei-Shaw/sub2api/internal/ccgo/shellwrapper"
	"github.com/stretchr/testify/require"
)

type execRequesterFunc func(context.Context, int64, protocol.ExecRequest) (protocol.ExecResponse, error)

func (f execRequesterFunc) Exec(ctx context.Context, workspaceID int64, req protocol.ExecRequest) (protocol.ExecResponse, error) {
	return f(ctx, workspaceID, req)
}

func TestExecBridgeRoutesWrapperRequestToHubExec(t *testing.T) {
	socketPath := shortSocketPath(t, "exec-1.sock")
	requestCh := make(chan protocol.ExecRequest, 1)
	bridge, err := StartExecBridge(socketPath, execRequesterFunc(func(ctx context.Context, workspaceID int64, req protocol.ExecRequest) (protocol.ExecResponse, error) {
		require.Equal(t, int64(77), workspaceID)
		requestCh <- req
		return protocol.ExecResponse{Stdout: "ok\n", Stderr: "warn\n", ExitCode: 3}, nil
	}))
	require.NoError(t, err)
	defer bridge.Close()

	conn, err := net.Dial("unix", socketPath)
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, json.NewEncoder(conn).Encode(shellwrapper.ExecBridgeRequest{
		WorkspaceID: 77,
		Cwd:         "/Users/alice/project",
		Command:     "npm test",
		TimeoutMS:   1000,
	}))
	var resp shellwrapper.ExecBridgeResponse
	require.NoError(t, json.NewDecoder(conn).Decode(&resp))

	require.Equal(t, "ok\n", resp.Stdout)
	require.Equal(t, "warn\n", resp.Stderr)
	require.Equal(t, 3, resp.ExitCode)
	require.Nil(t, resp.Error)
	req := <-requestCh
	require.Equal(t, "/Users/alice/project", req.Cwd)
	require.Equal(t, "npm test", req.Command)
	require.Equal(t, int64(1000), req.TimeoutMS)
}

func TestExecBridgeReturnsAgentError(t *testing.T) {
	socketPath := shortSocketPath(t, "exec-2.sock")
	bridge, err := StartExecBridge(socketPath, execRequesterFunc(func(context.Context, int64, protocol.ExecRequest) (protocol.ExecResponse, error) {
		return protocol.ExecResponse{}, protocol.NewError(protocol.ErrorAgentDisconnected, "agent is not connected")
	}))
	require.NoError(t, err)
	defer bridge.Close()

	conn, err := net.Dial("unix", socketPath)
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, json.NewEncoder(conn).Encode(shellwrapper.ExecBridgeRequest{
		WorkspaceID: 77,
		Cwd:         "/Users/alice/project",
		Command:     "npm test",
	}))
	var resp shellwrapper.ExecBridgeResponse
	require.NoError(t, json.NewDecoder(conn).Decode(&resp))
	require.Equal(t, 1, resp.ExitCode)
	require.NotNil(t, resp.Error)
	require.Equal(t, protocol.ErrorAgentDisconnected, resp.Error.Code)
}

func shortSocketPath(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(os.TempDir(), "ccgo-runner-test")
	require.NoError(t, os.MkdirAll(dir, 0o700))
	path := filepath.Join(dir, name)
	t.Cleanup(func() { _ = os.Remove(path) })
	return path
}
