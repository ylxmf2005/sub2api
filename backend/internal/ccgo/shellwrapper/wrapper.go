package shellwrapper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/protocol"
)

const (
	EnvWorkspaceID = "CCGO_WORKSPACE_ID"
	EnvServerRoot  = "CCGO_SERVER_ROOT"
	EnvLocalRoot   = "CCGO_LOCAL_ROOT"
	EnvExecSocket  = "CCGO_EXEC_SOCKET"
	EnvExecTimeout = "CCGO_EXEC_TIMEOUT_MS"
)

type ExecBridgeRequest struct {
	WorkspaceID int64  `json:"workspace_id"`
	Cwd         string `json:"cwd"`
	Command     string `json:"command"`
	TimeoutMS   int64  `json:"timeout_ms,omitempty"`
}

type ExecBridgeResponse struct {
	Stdout   string          `json:"stdout,omitempty"`
	Stderr   string          `json:"stderr,omitempty"`
	ExitCode int             `json:"exit_code"`
	Error    *protocol.Error `json:"error,omitempty"`
}

type Runner struct {
	Stdout io.Writer
	Stderr io.Writer
	Getwd  func() (string, error)
	Dial   func(context.Context, string) (net.Conn, error)
	Env    func(string) string
	Args   []string
}

func (r Runner) Run(ctx context.Context) int {
	stdout := r.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := r.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}
	env := r.Env
	if env == nil {
		env = os.Getenv
	}
	command := strings.TrimSpace(strings.Join(r.Args, " "))
	if command == "" {
		_, _ = fmt.Fprintln(stderr, "ccgo-wrapper: command is required")
		return 2
	}
	workspaceID, err := strconv.ParseInt(strings.TrimSpace(env(EnvWorkspaceID)), 10, 64)
	if err != nil || workspaceID <= 0 {
		_, _ = fmt.Fprintln(stderr, "ccgo-wrapper: CCGO_WORKSPACE_ID is required")
		return 2
	}
	mapper, err := NewPathMapper(env(EnvServerRoot), env(EnvLocalRoot))
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ccgo-wrapper: %v\n", err)
		return 2
	}
	getwd := r.Getwd
	if getwd == nil {
		getwd = os.Getwd
	}
	serverCwd, err := getwd()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ccgo-wrapper: read cwd: %v\n", err)
		return 2
	}
	localCwd, err := mapper.MapCwd(serverCwd)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ccgo-wrapper: %v\n", err)
		return 2
	}
	socketPath := strings.TrimSpace(env(EnvExecSocket))
	if socketPath == "" {
		_, _ = fmt.Fprintln(stderr, "ccgo-wrapper: CCGO_EXEC_SOCKET is required")
		return 2
	}
	timeoutMS := parseTimeoutMS(env(EnvExecTimeout))
	resp, err := r.exec(ctx, socketPath, ExecBridgeRequest{
		WorkspaceID: workspaceID,
		Cwd:         localCwd,
		Command:     mapper.MapCommand(command),
		TimeoutMS:   timeoutMS,
	})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ccgo-wrapper: %v\n", err)
		return 1
	}
	if resp.Stdout != "" {
		_, _ = io.WriteString(stdout, resp.Stdout)
	}
	if resp.Stderr != "" {
		_, _ = io.WriteString(stderr, resp.Stderr)
	}
	if resp.Error != nil {
		if resp.Error.Message != "" {
			_, _ = fmt.Fprintf(stderr, "ccgo-wrapper: %s\n", resp.Error.Error())
		}
		if resp.ExitCode != 0 {
			return resp.ExitCode
		}
		return 1
	}
	return resp.ExitCode
}

func (r Runner) exec(ctx context.Context, socketPath string, req ExecBridgeRequest) (ExecBridgeResponse, error) {
	dial := r.Dial
	if dial == nil {
		var d net.Dialer
		dial = func(ctx context.Context, networkAddr string) (net.Conn, error) {
			return d.DialContext(ctx, "unix", networkAddr)
		}
	}
	conn, err := dial(ctx, socketPath)
	if err != nil {
		return ExecBridgeResponse{}, fmt.Errorf("connect ccgo exec bridge: %w", err)
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return ExecBridgeResponse{}, fmt.Errorf("send ccgo exec request: %w", err)
	}
	var resp ExecBridgeResponse
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return ExecBridgeResponse{}, fmt.Errorf("read ccgo exec response: %w", err)
	}
	return resp, nil
}

func parseTimeoutMS(value string) int64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed < 0 {
		return 0
	}
	if parsed > int64((24*time.Hour)/time.Millisecond) {
		return int64((24 * time.Hour) / time.Millisecond)
	}
	return parsed
}
