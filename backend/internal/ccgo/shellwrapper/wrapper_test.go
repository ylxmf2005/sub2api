package shellwrapper

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/protocol"
	"github.com/stretchr/testify/require"
)

func TestRunnerSendsMappedExecRequestAndReturnsExitCode(t *testing.T) {
	var stderr bytes.Buffer
	var stdout bytes.Buffer
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	requestCh := make(chan ExecBridgeRequest, 1)
	go func() {
		defer server.Close()
		var req ExecBridgeRequest
		require.NoError(t, json.NewDecoder(server).Decode(&req))
		requestCh <- req
		require.NoError(t, json.NewEncoder(server).Encode(ExecBridgeResponse{
			Stdout:   "out\n",
			Stderr:   "err\n",
			ExitCode: 7,
		}))
	}()

	code := Runner{
		Args:   []string{"cat", "/srv/ccgo/ws/project/package.json"},
		Stdout: &stdout,
		Stderr: &stderr,
		Getwd:  func() (string, error) { return "/srv/ccgo/ws/project", nil },
		Dial:   func(context.Context, string) (net.Conn, error) { return client, nil },
		Env: func(key string) string {
			switch key {
			case EnvWorkspaceID:
				return "42"
			case EnvServerRoot:
				return "/srv/ccgo/ws/project"
			case EnvLocalRoot:
				return "/Users/alice/project"
			case EnvExecSocket:
				return "/tmp/ccgo.sock"
			default:
				return ""
			}
		},
	}.Run(context.Background())

	require.Equal(t, 7, code)
	require.Equal(t, "out\n", stdout.String())
	require.Equal(t, "err\n", stderr.String())
	req := <-requestCh
	require.Equal(t, int64(42), req.WorkspaceID)
	require.Equal(t, "/Users/alice/project", req.Cwd)
	require.Equal(t, "cat '/Users/alice/project/package.json'", req.Command)
}

func TestRunnerFailsClosedWhenExecBridgeMissing(t *testing.T) {
	var stderr bytes.Buffer

	code := Runner{
		Args:   []string{"pwd"},
		Stderr: &stderr,
		Getwd:  func() (string, error) { return "/srv/ccgo/ws/project", nil },
		Env: func(key string) string {
			switch key {
			case EnvWorkspaceID:
				return "42"
			case EnvServerRoot:
				return "/srv/ccgo/ws/project"
			case EnvLocalRoot:
				return "/Users/alice/project"
			default:
				return ""
			}
		},
	}.Run(context.Background())

	require.Equal(t, 2, code)
	require.Contains(t, stderr.String(), EnvExecSocket)
}

func TestRunnerReportsAgentErrorWithoutServerFallback(t *testing.T) {
	var stderr bytes.Buffer
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	go func() {
		defer server.Close()
		var req ExecBridgeRequest
		require.NoError(t, json.NewDecoder(server).Decode(&req))
		require.NoError(t, json.NewEncoder(server).Encode(ExecBridgeResponse{
			ExitCode: 1,
			Error:    protocol.NewError(protocol.ErrorAgentDisconnected, "agent is not connected"),
		}))
	}()

	code := Runner{
		Args:   []string{"npm", "test"},
		Stderr: &stderr,
		Getwd:  func() (string, error) { return "/srv/ccgo/ws/project", nil },
		Dial:   func(context.Context, string) (net.Conn, error) { return client, nil },
		Env: func(key string) string {
			switch key {
			case EnvWorkspaceID:
				return "42"
			case EnvServerRoot:
				return "/srv/ccgo/ws/project"
			case EnvLocalRoot:
				return "/Users/alice/project"
			case EnvExecSocket:
				return "/tmp/ccgo.sock"
			default:
				return ""
			}
		},
	}.Run(context.Background())

	require.Equal(t, 1, code)
	require.Contains(t, stderr.String(), protocol.ErrorAgentDisconnected)
}
