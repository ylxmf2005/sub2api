package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/shellwrapper"
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
