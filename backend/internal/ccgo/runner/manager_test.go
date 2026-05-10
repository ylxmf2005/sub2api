package runner

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/projection"
	"github.com/Wei-Shaw/sub2api/internal/ccgo/protocol"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type runnerRepoStub struct {
	workspace *service.CcgoWorkspace
	activeRun *service.CcgoWorkstationRun
	latestRun *service.CcgoWorkstationRun
	created   *service.CcgoWorkstationRun
	running   *service.CcgoWorkstationRun
	stopped   *service.CcgoWorkstationRun
	audits    []service.CcgoCommandAuditInput
	failed    bool
}

func (r *runnerRepoStub) ResolveWorkspace(context.Context, service.CcgoResolveWorkspaceInput) (*service.CcgoWorkspaceResolution, error) {
	return nil, service.ErrCcgoWorkspaceUnavailable
}
func (r *runnerRepoStub) CreateDeviceLogin(context.Context, string, string, string, time.Time) (*service.CcgoDeviceLogin, error) {
	return nil, service.ErrCcgoWorkspaceUnavailable
}
func (r *runnerRepoStub) FindDeviceLoginByDeviceCodeHash(context.Context, string) (*service.CcgoDeviceLogin, error) {
	return nil, service.ErrCcgoDeviceLoginNotFound
}
func (r *runnerRepoStub) FindDeviceLoginByUserCodeHash(context.Context, string) (*service.CcgoDeviceLogin, error) {
	return nil, service.ErrCcgoDeviceLoginNotFound
}
func (r *runnerRepoStub) ApproveDeviceLogin(context.Context, int64, int64, time.Time) (*service.CcgoDeviceLogin, error) {
	return nil, service.ErrCcgoDeviceLoginNotFound
}
func (r *runnerRepoStub) ConsumeDeviceLogin(context.Context, int64, time.Time) (*service.CcgoDeviceLogin, error) {
	return nil, service.ErrCcgoDeviceLoginNotFound
}
func (r *runnerRepoStub) ExpireDeviceLogin(context.Context, int64, time.Time) (*service.CcgoDeviceLogin, error) {
	return nil, service.ErrCcgoDeviceLoginNotFound
}
func (r *runnerRepoStub) IssueAgentCredential(context.Context, *service.CcgoWorkspace, time.Duration) (*service.CcgoIssuedCredential, error) {
	return nil, service.ErrCcgoWorkspaceUnavailable
}
func (r *runnerRepoStub) FindAgentCredentialByTokenHash(context.Context, string) (*service.CcgoAgentCredential, error) {
	return nil, service.ErrCcgoAgentCredentialInvalid
}
func (r *runnerRepoStub) MarkAgentCredentialUsed(context.Context, int64) error {
	return service.ErrCcgoAgentCredentialInvalid
}
func (r *runnerRepoStub) GetWorkspace(context.Context, int64) (*service.CcgoWorkspace, error) {
	return r.workspace, nil
}
func (r *runnerRepoStub) FindActiveWorkstationRun(context.Context, int64) (*service.CcgoWorkstationRun, error) {
	return r.activeRun, nil
}
func (r *runnerRepoStub) FindLatestWorkstationRun(context.Context, int64) (*service.CcgoWorkstationRun, error) {
	if r.latestRun != nil {
		return r.latestRun, nil
	}
	return r.activeRun, nil
}
func (r *runnerRepoStub) CreateWorkstationRun(context.Context, *service.CcgoWorkspace, string, time.Time) (*service.CcgoWorkstationRun, error) {
	r.created = &service.CcgoWorkstationRun{WorkspaceID: r.workspace.ID, UserID: r.workspace.UserID, RunID: "run_test", Status: service.CcgoRunStatusStarting}
	return r.created, nil
}
func (r *runnerRepoStub) MarkWorkstationRunRunning(context.Context, string, string, time.Time) (*service.CcgoWorkstationRun, error) {
	r.running = &service.CcgoWorkstationRun{WorkspaceID: r.workspace.ID, UserID: r.workspace.UserID, RunID: "run_test", Status: service.CcgoRunStatusRunning, ServerPID: "1234"}
	return r.running, nil
}
func (r *runnerRepoStub) MarkWorkstationRunStopped(context.Context, string, string, time.Time) (*service.CcgoWorkstationRun, error) {
	r.stopped = &service.CcgoWorkstationRun{WorkspaceID: r.workspace.ID, UserID: r.workspace.UserID, RunID: "run_test", Status: service.CcgoRunStatusStopped}
	return r.stopped, nil
}
func (r *runnerRepoStub) MarkWorkstationRunFailed(context.Context, string, string, time.Time) error {
	r.failed = true
	return nil
}
func (r *runnerRepoStub) CreateCommandAudit(_ context.Context, input service.CcgoCommandAuditInput) error {
	r.audits = append(r.audits, input)
	return nil
}

type runnerRequesterStub struct{}

func (runnerRequesterStub) Exec(context.Context, int64, protocol.ExecRequest) (protocol.ExecResponse, error) {
	return protocol.ExecResponse{}, nil
}
func (runnerRequesterStub) FileStat(context.Context, int64, protocol.FileStatRequest) (protocol.FileStatResponse, error) {
	return protocol.FileStatResponse{}, nil
}
func (runnerRequesterStub) FileRead(context.Context, int64, protocol.FileReadRequest) (protocol.FileReadResponse, error) {
	return protocol.FileReadResponse{}, nil
}
func (runnerRequesterStub) FileWrite(context.Context, int64, protocol.FileWriteRequest) (protocol.FileWriteResponse, error) {
	return protocol.FileWriteResponse{}, nil
}
func (runnerRequesterStub) FileList(context.Context, int64, protocol.FileListRequest) (protocol.FileListResponse, error) {
	return protocol.FileListResponse{}, nil
}
func (runnerRequesterStub) FileMkdir(context.Context, int64, protocol.FileMkdirRequest) (protocol.FileStatResponse, error) {
	return protocol.FileStatResponse{}, nil
}
func (runnerRequesterStub) FileRemove(context.Context, int64, protocol.FileRemoveRequest) error {
	return nil
}
func (runnerRequesterStub) FileRename(context.Context, int64, protocol.FileRenameRequest) error {
	return nil
}
func (runnerRequesterStub) FileTruncate(context.Context, int64, protocol.FileTruncateRequest) (protocol.FileStatResponse, error) {
	return protocol.FileStatResponse{}, nil
}
func (runnerRequesterStub) FileChmod(context.Context, int64, protocol.FileChmodRequest) (protocol.FileStatResponse, error) {
	return protocol.FileStatResponse{}, nil
}
func (runnerRequesterStub) FileReadlink(context.Context, int64, protocol.FileReadlinkRequest) (protocol.FileReadlinkResponse, error) {
	return protocol.FileReadlinkResponse{}, nil
}

type fakeMount struct {
	unmounted bool
}

func (m *fakeMount) Unmount() error {
	m.unmounted = true
	return nil
}

type fakePTYStarter struct {
	started *exec.Cmd
	pty     *memoryPTY
}

func (s *fakePTYStarter) Start(cmd *exec.Cmd, cols, rows uint16) (PTY, error) {
	s.started = cmd
	if s.pty == nil {
		s.pty = newMemoryPTY()
	}
	cmd.Process = &os.Process{Pid: 1234}
	return s.pty, nil
}

type memoryPTY struct {
	mu     sync.Mutex
	buf    bytes.Buffer
	readCh chan []byte
	closed chan struct{}
}

func newMemoryPTY() *memoryPTY {
	return &memoryPTY{readCh: make(chan []byte, 8), closed: make(chan struct{})}
}

func (p *memoryPTY) Read(data []byte) (int, error) {
	select {
	case chunk := <-p.readCh:
		return copy(data, chunk), nil
	case <-p.closed:
		return 0, io.EOF
	}
}

func (p *memoryPTY) Write(data []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.buf.Write(data)
}

func (p *memoryPTY) Close() error {
	select {
	case <-p.closed:
	default:
		close(p.closed)
	}
	return nil
}

func (p *memoryPTY) Resize(uint16, uint16) error {
	return nil
}

func TestManagerStartCcgoRunLaunchesProjectionBridgeAndPTY(t *testing.T) {
	workspace := &service.CcgoWorkspace{
		ID:               42,
		UserID:           7,
		ServerRoot:       "/srv/ccgo/workspaces/u7/project",
		LocalRootDisplay: "/Users/alice/project",
		PathStyle:        service.CcgoPathStylePOSIX,
	}
	repo := &runnerRepoStub{workspace: workspace}
	starter := &fakePTYStarter{}
	mount := &fakeMount{}
	manager := NewManager(repo, runnerRequesterStub{})
	manager.PTYStarter = starter
	manager.RuntimeBase = shortSocketDir(t)
	manager.ConfigBase = t.TempDir()
	manager.ClaudeBinary = "/bin/claude-test"
	manager.WrapperBinary = "/bin/ccgo-wrapper-test"
	manager.DisableAutoCleanup = true
	manager.Mount = func(dir string, backend projection.Backend) (interface{ Unmount() error }, error) {
		require.Equal(t, workspace.ServerRoot, dir)
		require.NotNil(t, backend)
		return mount, nil
	}

	run, reused, err := manager.StartCcgoRun(context.Background(), workspace)
	require.NoError(t, err)
	require.False(t, reused)
	require.Equal(t, service.CcgoRunStatusRunning, run.Status)
	require.NotNil(t, starter.started)
	require.Equal(t, workspace.ServerRoot, starter.started.Dir)
	require.Contains(t, starter.started.Args, "--append-system-prompt-file")
	require.NotEmpty(t, envValue(starter.started.Env, "CCGO_EXEC_SOCKET"))
	require.False(t, mount.unmounted)
}

func TestManagerStopCcgoRunCleansRuntimeAndMarksRunStopped(t *testing.T) {
	workspace := &service.CcgoWorkspace{
		ID:               42,
		UserID:           7,
		ServerRoot:       "/srv/ccgo/workspaces/u7/project",
		LocalRootDisplay: "/Users/alice/project",
		PathStyle:        service.CcgoPathStylePOSIX,
	}
	repo := &runnerRepoStub{workspace: workspace}
	starter := &fakePTYStarter{}
	mount := &fakeMount{}
	manager := NewManager(repo, runnerRequesterStub{})
	manager.PTYStarter = starter
	manager.RuntimeBase = shortSocketDir(t)
	manager.ConfigBase = t.TempDir()
	manager.ClaudeBinary = "/bin/claude-test"
	manager.WrapperBinary = "/bin/ccgo-wrapper-test"
	manager.DisableAutoCleanup = true
	manager.Mount = func(string, projection.Backend) (interface{ Unmount() error }, error) {
		return mount, nil
	}
	_, _, err := manager.StartCcgoRun(context.Background(), workspace)
	require.NoError(t, err)

	stopped, status, err := manager.StopCcgoRun(context.Background(), workspace.ID, "test stop")
	require.NoError(t, err)
	require.NotNil(t, stopped)
	require.Equal(t, service.CcgoRunStatusStopped, stopped.Status)
	require.False(t, status.Running)
	require.False(t, status.ProjectionMounted)
	require.True(t, mount.unmounted)
	_, ok := manager.Store.Get(workspace.ID)
	require.False(t, ok)
}

func shortSocketDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(os.TempDir(), "ccgo-runner")
	require.NoError(t, os.MkdirAll(dir, 0o700))
	return dir
}

func TestManagerStartCcgoRunReusesActiveRun(t *testing.T) {
	workspace := &service.CcgoWorkspace{ID: 42, UserID: 7}
	repo := &runnerRepoStub{
		workspace: workspace,
		activeRun: &service.CcgoWorkstationRun{WorkspaceID: 42, UserID: 7, RunID: "run_existing", Status: service.CcgoRunStatusRunning},
	}
	manager := NewManager(repo, runnerRequesterStub{})
	manager.PTYStarter = &fakePTYStarter{}

	run, reused, err := manager.StartCcgoRun(context.Background(), workspace)
	require.NoError(t, err)
	require.True(t, reused)
	require.Equal(t, "run_existing", run.RunID)
	require.Nil(t, repo.created)
}
