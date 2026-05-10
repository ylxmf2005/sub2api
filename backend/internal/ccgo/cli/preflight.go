package cli

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const localPreflightProbe = "ccgo-preflight"

type LocalPreflightResult struct {
	LocalRoot  string `json:"local_root"`
	OS         string `json:"os"`
	PathStyle  string `json:"path_style"`
	ShellName  string `json:"shell_name"`
	ShellPath  string `json:"shell_path"`
	ProbeValue string `json:"probe_value"`
}

type localPreflightOptions struct {
	goos          string
	lookPath      func(string) (string, error)
	resolvePath   func(string) (*LocalPathInfo, error)
	runShellProbe func(context.Context, string, string) (string, error)
}

func CheckLocalPreflight(ctx context.Context, localPath string) (*LocalPreflightResult, error) {
	return checkLocalPreflight(ctx, localPath, localPreflightOptions{})
}

func checkLocalPreflight(ctx context.Context, localPath string, opts localPreflightOptions) (*LocalPreflightResult, error) {
	resolvePath := opts.resolvePath
	if resolvePath == nil {
		resolvePath = ResolveLocalPath
	}
	info, err := resolvePath(localPath)
	if err != nil {
		return nil, fmt.Errorf("ccgo local preflight failed: %w", err)
	}

	goos := strings.TrimSpace(opts.goos)
	if goos == "" {
		goos = runtime.GOOS
	}
	lookPath := opts.lookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	shellPath, err := lookPath("bash")
	if err != nil {
		return nil, missingBashPreflightError(goos, err)
	}

	runShellProbe := opts.runShellProbe
	if runShellProbe == nil {
		runShellProbe = runBashPreflightProbe
	}
	probeValue, err := runShellProbe(ctx, shellPath, info.CanonicalRoot)
	if err != nil {
		return nil, fmt.Errorf("ccgo local preflight failed: bash exists but cannot run in the local project root: %w", err)
	}
	probeValue = strings.TrimSpace(probeValue)
	if probeValue != localPreflightProbe {
		return nil, fmt.Errorf("ccgo local preflight failed: bash probe returned %q", probeValue)
	}

	return &LocalPreflightResult{
		LocalRoot:  info.CanonicalRoot,
		OS:         info.OS,
		PathStyle:  info.PathStyle,
		ShellName:  "bash",
		ShellPath:  shellPath,
		ProbeValue: probeValue,
	}, nil
}

func runBashPreflightProbe(ctx context.Context, shellPath string, localRoot string) (string, error) {
	probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(probeCtx, shellPath, "-lc", "printf "+localPreflightProbe)
	cmd.Dir = localRoot
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if probeCtx.Err() != nil {
		return "", probeCtx.Err()
	}
	if err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = strings.TrimSpace(stdout.String())
		}
		if detail == "" {
			return "", err
		}
		return "", fmt.Errorf("%w: %s", err, detail)
	}
	return stdout.String(), nil
}

func missingBashPreflightError(goos string, cause error) error {
	if goos == "windows" {
		return fmt.Errorf("ccgo local preflight failed: bash is required for MVP command execution on Windows; install Git Bash and put bash.exe on PATH, or run ccgo from WSL: %w", cause)
	}
	return fmt.Errorf("ccgo local preflight failed: bash is required for local command execution: %w", cause)
}
