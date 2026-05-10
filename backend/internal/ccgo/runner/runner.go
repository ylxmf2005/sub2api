package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/projection"
	"github.com/Wei-Shaw/sub2api/internal/ccgo/protocol"
	"github.com/Wei-Shaw/sub2api/internal/ccgo/shellwrapper"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type LaunchConfig struct {
	WorkspaceID      int64
	ServerRoot       string
	LocalRootDisplay string
	PathStyle        string
	ConfigBaseDir    string
	RuntimeBaseDir   string
	ClaudeBinary     string
	WrapperBinary    string
	Args             []string
	Env              map[string]string
}

type LaunchSpec struct {
	WorkspaceID int64
	Dir         string
	Binary      string
	Args        []string
	Env         []string
	ConfigDir   string
	PromptFile  string
	ExecSocket  string
}

func BuildLaunchSpec(cfg LaunchConfig) (*LaunchSpec, error) {
	if cfg.WorkspaceID <= 0 {
		return nil, fmt.Errorf("ccgo workspace id is required")
	}
	serverRoot := strings.TrimSpace(cfg.ServerRoot)
	if serverRoot == "" {
		return nil, fmt.Errorf("ccgo server root is required")
	}
	localRoot := strings.TrimSpace(cfg.LocalRootDisplay)
	if localRoot == "" {
		return nil, fmt.Errorf("ccgo local root is required")
	}
	claudeBinary := strings.TrimSpace(cfg.ClaudeBinary)
	if claudeBinary == "" {
		claudeBinary = "claude"
	}
	wrapperBinary := strings.TrimSpace(cfg.WrapperBinary)
	if wrapperBinary == "" {
		wrapperBinary = "ccgo-wrapper"
	}
	configBase := strings.TrimSpace(cfg.ConfigBaseDir)
	if configBase == "" {
		configBase = filepath.Join(os.TempDir(), "ccgo", "claude-config")
	}
	runtimeBase := strings.TrimSpace(cfg.RuntimeBaseDir)
	if runtimeBase == "" {
		runtimeBase = filepath.Join(os.TempDir(), "ccgo", "runtime")
	}
	workspaceKey := "workspace-" + strconv.FormatInt(cfg.WorkspaceID, 10)
	configDir := filepath.Join(configBase, workspaceKey)
	runtimeDir := filepath.Join(runtimeBase, workspaceKey)
	promptFile, err := WritePromptFile(runtimeDir, PromptConfig{
		LocalRootDisplay: localRoot,
		ServerRoot:       serverRoot,
		PathStyle:        cfg.PathStyle,
	})
	if err != nil {
		return nil, err
	}
	execSocket := filepath.Join(runtimeDir, "exec.sock")
	args := append([]string{"--append-system-prompt-file", promptFile}, cfg.Args...)
	env := append(os.Environ(),
		"CLAUDE_CONFIG_DIR="+configDir,
		"CLAUDE_CODE_SHELL_PREFIX="+wrapperBinary,
		shellwrapper.EnvWorkspaceID+"="+strconv.FormatInt(cfg.WorkspaceID, 10),
		shellwrapper.EnvServerRoot+"="+serverRoot,
		shellwrapper.EnvLocalRoot+"="+localRoot,
		shellwrapper.EnvExecSocket+"="+execSocket,
	)
	for key, value := range cfg.Env {
		key = strings.TrimSpace(key)
		if key == "" || strings.Contains(key, "=") {
			return nil, fmt.Errorf("invalid ccgo runner env key %q", key)
		}
		env = append(env, key+"="+value)
	}
	return &LaunchSpec{
		WorkspaceID: cfg.WorkspaceID,
		Dir:         serverRoot,
		Binary:      claudeBinary,
		Args:        args,
		Env:         env,
		ConfigDir:   configDir,
		PromptFile:  promptFile,
		ExecSocket:  execSocket,
	}, nil
}

type ProcessLauncher interface {
	Start(context.Context, *LaunchSpec) (*exec.Cmd, error)
}

type CommandLauncher struct{}

func (CommandLauncher) Start(ctx context.Context, spec *LaunchSpec) (*exec.Cmd, error) {
	if spec == nil {
		return nil, fmt.Errorf("ccgo launch spec is required")
	}
	cmd := exec.CommandContext(ctx, spec.Binary, spec.Args...)
	cmd.Dir = spec.Dir
	cmd.Env = spec.Env
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start Claude Code: %w", err)
	}
	return cmd, nil
}

type MountFunc func(string, projection.Backend) (interface{ Unmount() error }, error)

type Manager struct {
	Repo               service.CcgoRepository
	Requester          WorkspaceRequester
	Store              *ProcessStore
	PTYStarter         PTYStarter
	Mount              MountFunc
	ConfigBase         string
	RuntimeBase        string
	ClaudeBinary       string
	WrapperBinary      string
	DisableAutoCleanup bool
}

func NewManager(repo service.CcgoRepository, requester WorkspaceRequester) *Manager {
	return &Manager{
		Repo:       repo,
		Requester:  requester,
		Store:      NewProcessStore(),
		PTYStarter: OSPTYStarter{},
		Mount: func(dir string, backend projection.Backend) (interface{ Unmount() error }, error) {
			return projection.MountWorkspace(dir, backend)
		},
	}
}

func ProvideManager(repo service.CcgoRepository, requester WorkspaceRequester) *Manager {
	return NewManager(repo, requester)
}

func ProvideRunStarter(manager *Manager) service.CcgoRunStarter {
	if manager == nil {
		return nil
	}
	return manager
}

func ProvideTerminalAttacher(manager *Manager) service.CcgoTerminalAttacher {
	if manager == nil {
		return nil
	}
	return manager
}

func ProvideRunnerController(manager *Manager) service.CcgoRunnerController {
	if manager == nil {
		return nil
	}
	return manager
}

func (m *Manager) StartCcgoRun(ctx context.Context, workspace *service.CcgoWorkspace) (*service.CcgoWorkstationRun, bool, error) {
	if workspace == nil || workspace.ID <= 0 {
		return nil, false, fmt.Errorf("ccgo workspace is required")
	}
	if m == nil || m.Repo == nil || m.Requester == nil {
		return nil, false, fmt.Errorf("ccgo runner is not configured")
	}
	if m.Store == nil {
		m.Store = NewProcessStore()
	}
	if existing, ok := m.Store.Get(workspace.ID); ok {
		run, err := m.Repo.FindActiveWorkstationRun(ctx, workspace.ID)
		if err != nil {
			return nil, false, err
		}
		if run != nil {
			return run, true, nil
		}
		m.Store.Delete(existing.WorkspaceID)
	}
	if run, err := m.Repo.FindActiveWorkstationRun(ctx, workspace.ID); err != nil {
		return nil, false, err
	} else if run != nil {
		return run, true, nil
	}

	now := timeNow()
	runID := "run_" + strings.ReplaceAll(strconv.FormatInt(now.UnixNano(), 36), "-", "")
	run, err := m.Repo.CreateWorkstationRun(ctx, workspace, runID, now)
	if err != nil {
		return nil, false, err
	}
	session, err := m.launchSession(ctx, workspace, run)
	if err != nil {
		_ = m.Repo.MarkWorkstationRunFailed(ctx, run.RunID, err.Error(), timeNow())
		return nil, false, err
	}
	if err := m.Store.Put(session); err != nil {
		_ = session.Stop()
		_ = m.Repo.MarkWorkstationRunFailed(ctx, run.RunID, err.Error(), timeNow())
		return nil, false, err
	}
	running, err := m.Repo.MarkWorkstationRunRunning(ctx, run.RunID, strconv.Itoa(session.Cmd.Process.Pid), timeNow())
	if err != nil {
		_ = session.Stop()
		m.Store.Delete(workspace.ID)
		return nil, false, err
	}
	if !m.DisableAutoCleanup {
		go m.waitAndCleanup(run.RunID, workspace.ID, session)
	}
	return running, false, nil
}

func (m *Manager) Attach(workspaceID int64, input io.Reader, output io.Writer) error {
	if m == nil || m.Store == nil {
		return fmt.Errorf("ccgo runner is not configured")
	}
	session, ok := m.Store.Get(workspaceID)
	if !ok {
		return fmt.Errorf("ccgo workstation is not running")
	}
	return session.Attach(input, output)
}

func (m *Manager) RuntimeStatus(ctx context.Context, workspaceID int64) (*service.CcgoRunnerStatus, error) {
	if workspaceID <= 0 {
		return nil, fmt.Errorf("ccgo workspace id is required")
	}
	if m == nil || m.Store == nil {
		return &service.CcgoRunnerStatus{
			WorkspaceID:   workspaceID,
			LastCheckedAt: timeNow(),
		}, nil
	}
	session, ok := m.Store.Get(workspaceID)
	if !ok {
		return &service.CcgoRunnerStatus{
			WorkspaceID:   workspaceID,
			LastCheckedAt: timeNow(),
		}, nil
	}
	status, ok := session.RuntimeStatus()
	if !ok {
		return &service.CcgoRunnerStatus{
			WorkspaceID:   workspaceID,
			LastCheckedAt: timeNow(),
		}, nil
	}
	return status, nil
}

func (m *Manager) StopCcgoRun(ctx context.Context, workspaceID int64, reason string) (*service.CcgoWorkstationRun, *service.CcgoRunnerStatus, error) {
	if workspaceID <= 0 {
		return nil, nil, fmt.Errorf("ccgo workspace id is required")
	}
	if m == nil || m.Store == nil || m.Repo == nil {
		return nil, nil, fmt.Errorf("ccgo runner is not configured")
	}
	session, ok := m.Store.Take(workspaceID)
	status := &service.CcgoRunnerStatus{
		WorkspaceID:   workspaceID,
		LastCheckedAt: timeNow(),
	}
	if !ok {
		run, err := m.Repo.FindActiveWorkstationRun(ctx, workspaceID)
		if err != nil {
			return nil, status, err
		}
		if run == nil {
			return nil, status, nil
		}
		stopped, err := m.Repo.MarkWorkstationRunStopped(ctx, run.RunID, normalizedStopReason(reason), timeNow())
		return stopped, status, err
	}
	if current, ok := session.RuntimeStatus(); ok {
		status = current
	}
	stopErr := session.Stop()
	status.Running = false
	status.ProjectionMounted = false
	status.ExecBridgeRunning = false
	status.TerminalReady = false
	status.LastCheckedAt = timeNow()
	if stopErr != nil {
		return nil, status, service.ErrCcgoRunnerCleanupFailed.WithCause(stopErr)
	}
	if session.RunID == "" {
		return nil, status, nil
	}
	stopped, err := m.Repo.MarkWorkstationRunStopped(ctx, session.RunID, normalizedStopReason(reason), timeNow())
	if err != nil {
		return nil, status, err
	}
	return stopped, status, nil
}

func (m *Manager) launchSession(ctx context.Context, workspace *service.CcgoWorkspace, run *service.CcgoWorkstationRun) (*Session, error) {
	starter := m.PTYStarter
	if starter == nil {
		starter = OSPTYStarter{}
	}
	mountFunc := m.Mount
	if mountFunc == nil {
		mountFunc = func(dir string, backend projection.Backend) (interface{ Unmount() error }, error) {
			return projection.MountWorkspace(dir, backend)
		}
	}
	backend := projection.NewHubBackend(workspace.ID, m.Requester)
	mount, err := mountFunc(workspace.ServerRoot, backend)
	if err != nil {
		return nil, fmt.Errorf("prepare ccgo projection: %w", err)
	}
	bridge, err := StartExecBridge(execSocketPath(m.RuntimeBase, workspace.ID), execRequesterWithAudit{
		requester: m.Requester,
		recorder:  &commandAuditRecorder{repo: m.Repo, workspace: workspace, run: run},
	})
	if err != nil {
		_ = mount.Unmount()
		return nil, err
	}
	spec, err := BuildLaunchSpec(LaunchConfig{
		WorkspaceID:      workspace.ID,
		ServerRoot:       workspace.ServerRoot,
		LocalRootDisplay: workspace.LocalRootDisplay,
		PathStyle:        workspace.PathStyle,
		ConfigBaseDir:    m.ConfigBase,
		RuntimeBaseDir:   m.RuntimeBase,
		ClaudeBinary:     m.ClaudeBinary,
		WrapperBinary:    m.WrapperBinary,
	})
	if err != nil {
		_ = bridge.Close()
		_ = mount.Unmount()
		return nil, err
	}
	cmd := exec.CommandContext(ctx, spec.Binary, spec.Args...)
	cmd.Dir = spec.Dir
	cmd.Env = overrideEnv(spec.Env, shellwrapper.EnvExecSocket, bridge.SocketPath())
	ptyHandle, err := starter.Start(cmd, 120, 40)
	if err != nil {
		_ = bridge.Close()
		_ = mount.Unmount()
		return nil, fmt.Errorf("start Claude Code PTY: %w", err)
	}
	return &Session{WorkspaceID: workspace.ID, RunID: run.RunID, Cmd: cmd, PTY: ptyHandle, Bridge: bridge, Mount: mount}, nil
}

func (m *Manager) waitAndCleanup(runID string, workspaceID int64, session *Session) {
	if session == nil || session.Cmd == nil {
		return
	}
	_ = session.Cmd.Wait()
	_ = session.Stop()
	if m.Store != nil {
		m.Store.Delete(workspaceID)
	}
	if m.Repo != nil && runID != "" {
		_, _ = m.Repo.MarkWorkstationRunStopped(context.Background(), runID, "process exited", timeNow())
	}
}

func timeNow() time.Time {
	return time.Now()
}

func execSocketPath(runtimeBase string, workspaceID int64) string {
	if strings.TrimSpace(runtimeBase) == "" {
		runtimeBase = filepath.Join(os.TempDir(), "ccgo", "runtime")
	}
	return filepath.Join(runtimeBase, "workspace-"+strconv.FormatInt(workspaceID, 10), "exec.sock")
}

func overrideEnv(env []string, key string, value string) []string {
	prefix := key + "="
	out := make([]string, 0, len(env)+1)
	replaced := false
	for _, item := range env {
		if strings.HasPrefix(item, prefix) {
			out = append(out, prefix+value)
			replaced = true
			continue
		}
		out = append(out, item)
	}
	if !replaced {
		out = append(out, prefix+value)
	}
	return out
}

func normalizedStopReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "stopped by user"
	}
	return reason
}

type WorkspaceRequester interface {
	ExecRequester
	FileStat(context.Context, int64, protocol.FileStatRequest) (protocol.FileStatResponse, error)
	FileRead(context.Context, int64, protocol.FileReadRequest) (protocol.FileReadResponse, error)
	FileWrite(context.Context, int64, protocol.FileWriteRequest) (protocol.FileWriteResponse, error)
	FileList(context.Context, int64, protocol.FileListRequest) (protocol.FileListResponse, error)
	FileMkdir(context.Context, int64, protocol.FileMkdirRequest) (protocol.FileStatResponse, error)
	FileRemove(context.Context, int64, protocol.FileRemoveRequest) error
	FileRename(context.Context, int64, protocol.FileRenameRequest) error
	FileTruncate(context.Context, int64, protocol.FileTruncateRequest) (protocol.FileStatResponse, error)
	FileChmod(context.Context, int64, protocol.FileChmodRequest) (protocol.FileStatResponse, error)
	FileReadlink(context.Context, int64, protocol.FileReadlinkRequest) (protocol.FileReadlinkResponse, error)
}

type execRequesterWithAudit struct {
	requester ExecRequester
	recorder  CommandAuditor
}

func (r execRequesterWithAudit) Exec(ctx context.Context, workspaceID int64, req protocol.ExecRequest) (protocol.ExecResponse, error) {
	return r.requester.Exec(ctx, workspaceID, req)
}

func (r execRequesterWithAudit) RecordCommandAudit(ctx context.Context, event CommandAuditEvent) error {
	if r.recorder == nil {
		return nil
	}
	return r.recorder.RecordCommandAudit(ctx, event)
}

type commandAuditRecorder struct {
	repo      service.CcgoRepository
	workspace *service.CcgoWorkspace
	run       *service.CcgoWorkstationRun
}

func (r *commandAuditRecorder) RecordCommandAudit(ctx context.Context, event CommandAuditEvent) error {
	if r == nil || r.repo == nil || r.workspace == nil {
		return nil
	}
	input := commandAuditInputFromEvent(r.workspace, r.run, event)
	return r.repo.CreateCommandAudit(ctx, input)
}

func commandAuditInputFromEvent(workspace *service.CcgoWorkspace, run *service.CcgoWorkstationRun, event CommandAuditEvent) service.CcgoCommandAuditInput {
	var runDatabaseID *int64
	if run != nil && run.ID > 0 {
		value := run.ID
		runDatabaseID = &value
	}
	exitCode := event.ExitCode
	status := service.CcgoCommandAuditStatusSucceeded
	failureReason := ""
	if event.Error != nil {
		failureReason = event.Error.Code
		switch event.Error.Code {
		case protocol.ErrorRequestTimeout:
			status = service.CcgoCommandAuditStatusTimeout
		case protocol.ErrorAgentDisconnected:
			status = service.CcgoCommandAuditStatusNotExecuted
		default:
			status = service.CcgoCommandAuditStatusFailed
		}
	} else if exitCode != 0 {
		status = service.CcgoCommandAuditStatusFailed
	}
	userID := int64(0)
	if workspace != nil {
		userID = workspace.UserID
	}
	return service.CcgoCommandAuditInput{
		WorkspaceID:     event.WorkspaceID,
		UserID:          userID,
		RunDatabaseID:   runDatabaseID,
		RequestID:       event.RequestID,
		CommandHash:     shellwrapper.HashCommand(event.Command),
		RedactedCommand: redactCommand(event.Command),
		ServerCwd:       event.ServerCwd,
		LocalCwd:        event.LocalCwd,
		ExitCode:        &exitCode,
		Status:          status,
		FailureReason:   failureReason,
		StartedAt:       event.StartedAt,
		FinishedAt:      event.FinishedAt,
		DurationMS:      event.FinishedAt.Sub(event.StartedAt).Milliseconds(),
	}
}

func redactCommand(command string) string {
	command = strings.TrimSpace(command)
	if command == "" {
		return ""
	}
	fields := strings.Fields(command)
	for i := 0; i < len(fields); i++ {
		lower := strings.ToLower(fields[i])
		switch lower {
		case "--token", "--password", "--secret", "--api-key":
			if i+1 < len(fields) {
				fields[i+1] = "[REDACTED]"
			}
			continue
		}
		for _, marker := range []string{"token=", "password=", "secret=", "api_key=", "apikey="} {
			if strings.Contains(lower, marker) {
				parts := strings.SplitN(fields[i], "=", 2)
				if len(parts) == 2 {
					fields[i] = parts[0] + "=[REDACTED]"
				}
				break
			}
		}
	}
	return strings.Join(fields, " ")
}
