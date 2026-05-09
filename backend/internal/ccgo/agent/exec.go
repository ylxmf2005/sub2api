package agent

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/protocol"
)

type ExecService struct {
	root LocalRoot
}

func NewExecService(root LocalRoot) *ExecService {
	return &ExecService{root: root}
}

func (s *ExecService) Run(ctx context.Context, req protocol.ExecRequest) (protocol.ExecResponse, error) {
	command := strings.TrimSpace(req.Command)
	if command == "" {
		return protocol.ExecResponse{}, protocol.NewError(protocol.ErrorInvalidRequest, "command is required")
	}
	cwd := req.Cwd
	if strings.TrimSpace(cwd) == "" {
		cwd = s.root.Path()
	}
	resolvedCwd, err := s.root.ResolveLocalCwd(cwd)
	if err != nil {
		return protocol.ExecResponse{}, err
	}
	timeout := time.Duration(req.TimeoutMS) * time.Millisecond
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, shellName(), shellArg(), command)
	cmd.Dir = resolvedCwd
	cmd.Env = execEnv(req.Env)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	response := protocol.ExecResponse{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}
	if ctx.Err() != nil {
		response.ExitCode = exitCodeFromError(err)
		return response, protocol.NewError(protocol.ErrorRequestTimeout, ctx.Err().Error())
	}
	if err != nil {
		response.ExitCode = exitCodeFromError(err)
		return response, nil
	}
	return response, nil
}

func shellName() string {
	return "bash"
}

func shellArg() string {
	return "-lc"
}

func execEnv(extra map[string]string) []string {
	env := os.Environ()
	env = append(env, "CCGO_LOCAL_EXEC=1")
	for key, value := range extra {
		key = strings.TrimSpace(key)
		if key == "" || strings.Contains(key, "=") {
			continue
		}
		env = append(env, key+"="+value)
	}
	return env
}

func exitCodeFromError(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return 1
}
