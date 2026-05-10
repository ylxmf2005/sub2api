package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/agent"
	"github.com/Wei-Shaw/sub2api/internal/ccgo/projection"
	"github.com/Wei-Shaw/sub2api/internal/ccgo/protocol"
	"github.com/Wei-Shaw/sub2api/internal/ccgo/runner"
	"github.com/Wei-Shaw/sub2api/internal/ccgo/shellwrapper"
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/stretchr/testify/require"
)

func TestReverseWorkstationHarnessUsesLocalFilesAndLocalExec(t *testing.T) {
	ctx := context.Background()
	localRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(localRoot, "README.md"), []byte("hello"), 0o644))

	localAgent, err := agent.New(localRoot)
	require.NoError(t, err)
	requester := agentRequester{agent: localAgent}

	projectionRoot := projection.NewRootNode(projection.NewHubBackend(42, requester))
	_ = fs.NewNodeFS(projectionRoot, nil)
	var entry fuse.EntryOut
	projectedFile, errno := projectionRoot.Lookup(ctx, "README.md", &entry)
	require.Equal(t, syscall.Errno(0), errno)
	fileNode, ok := projectedFile.Operations().(*projection.Node)
	require.True(t, ok)

	written, errno := fileNode.Write(ctx, nil, []byte(" from projection"), 5)
	require.Equal(t, syscall.Errno(0), errno)
	require.Equal(t, uint32(len(" from projection")), written)
	data, err := os.ReadFile(filepath.Join(localRoot, "README.md"))
	require.NoError(t, err)
	require.Equal(t, []byte("hello from projection"), data)

	socketPath := shortSocketPath(t, "reverse-workstation.sock")
	bridge, err := runner.StartExecBridge(socketPath, requester)
	require.NoError(t, err)
	defer bridge.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := shellwrapper.Runner{
		Args:   []string{"printf updated > README.md && pwd"},
		Stdout: &stdout,
		Stderr: &stderr,
		Getwd:  func() (string, error) { return "/srv/ccgo/workspaces/user/project", nil },
		Env: func(key string) string {
			switch key {
			case shellwrapper.EnvWorkspaceID:
				return "42"
			case shellwrapper.EnvServerRoot:
				return "/srv/ccgo/workspaces/user/project"
			case shellwrapper.EnvLocalRoot:
				return localRoot
			case shellwrapper.EnvExecSocket:
				return socketPath
			default:
				return ""
			}
		},
	}.Run(ctx)
	require.Equal(t, 0, exitCode, stderr.String())
	require.Empty(t, stderr.String())
	require.Contains(t, stdout.String(), localRoot)

	data, err = os.ReadFile(filepath.Join(localRoot, "README.md"))
	require.NoError(t, err)
	require.Equal(t, []byte("updated"), data)
}

type agentRequester struct {
	agent *agent.Agent
}

func (r agentRequester) request(ctx context.Context, method string, payload any, out any) error {
	env, err := protocol.NewRequest("req-test", method, payload)
	if err != nil {
		return err
	}
	resp, err := r.agent.Handle(ctx, env)
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return resp.Error
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(resp.Payload, out)
}

func (r agentRequester) FileStat(ctx context.Context, workspaceID int64, req protocol.FileStatRequest) (protocol.FileStatResponse, error) {
	var out protocol.FileStatResponse
	err := r.request(ctx, protocol.MethodFileStat, req, &out)
	return out, err
}

func (r agentRequester) FileRead(ctx context.Context, workspaceID int64, req protocol.FileReadRequest) (protocol.FileReadResponse, error) {
	var out protocol.FileReadResponse
	err := r.request(ctx, protocol.MethodFileRead, req, &out)
	return out, err
}

func (r agentRequester) FileWrite(ctx context.Context, workspaceID int64, req protocol.FileWriteRequest) (protocol.FileWriteResponse, error) {
	var out protocol.FileWriteResponse
	err := r.request(ctx, protocol.MethodFileWrite, req, &out)
	return out, err
}

func (r agentRequester) FileList(ctx context.Context, workspaceID int64, req protocol.FileListRequest) (protocol.FileListResponse, error) {
	var out protocol.FileListResponse
	err := r.request(ctx, protocol.MethodFileList, req, &out)
	return out, err
}

func (r agentRequester) FileMkdir(ctx context.Context, workspaceID int64, req protocol.FileMkdirRequest) (protocol.FileStatResponse, error) {
	var out protocol.FileStatResponse
	err := r.request(ctx, protocol.MethodFileMkdir, req, &out)
	return out, err
}

func (r agentRequester) FileRemove(ctx context.Context, workspaceID int64, req protocol.FileRemoveRequest) error {
	return r.request(ctx, protocol.MethodFileRemove, req, nil)
}

func (r agentRequester) FileRename(ctx context.Context, workspaceID int64, req protocol.FileRenameRequest) error {
	return r.request(ctx, protocol.MethodFileRename, req, nil)
}

func (r agentRequester) FileTruncate(ctx context.Context, workspaceID int64, req protocol.FileTruncateRequest) (protocol.FileStatResponse, error) {
	var out protocol.FileStatResponse
	err := r.request(ctx, protocol.MethodFileTruncate, req, &out)
	return out, err
}

func (r agentRequester) FileChmod(ctx context.Context, workspaceID int64, req protocol.FileChmodRequest) (protocol.FileStatResponse, error) {
	var out protocol.FileStatResponse
	err := r.request(ctx, protocol.MethodFileChmod, req, &out)
	return out, err
}

func (r agentRequester) FileReadlink(ctx context.Context, workspaceID int64, req protocol.FileReadlinkRequest) (protocol.FileReadlinkResponse, error) {
	var out protocol.FileReadlinkResponse
	err := r.request(ctx, protocol.MethodFileReadlink, req, &out)
	return out, err
}

func (r agentRequester) Exec(ctx context.Context, workspaceID int64, req protocol.ExecRequest) (protocol.ExecResponse, error) {
	var out protocol.ExecResponse
	err := r.request(ctx, protocol.MethodExec, req, &out)
	return out, err
}

func shortSocketPath(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(os.TempDir(), "ccgo-integration-test")
	require.NoError(t, os.MkdirAll(dir, 0o700))
	path := filepath.Join(dir, name)
	t.Cleanup(func() { _ = os.Remove(path) })
	return path
}
