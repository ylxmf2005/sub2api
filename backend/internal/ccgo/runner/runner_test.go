package runner

import (
	"os"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/shellwrapper"
	"github.com/stretchr/testify/require"
)

func TestBuildLaunchSpecInjectsClaudeCcgoEnvironment(t *testing.T) {
	tmp := t.TempDir()
	spec, err := BuildLaunchSpec(LaunchConfig{
		WorkspaceID:      123,
		ServerRoot:       "/srv/ccgo/workspaces/u1/project",
		LocalRootDisplay: "/Users/alice/project",
		PathStyle:        "posix",
		ConfigBaseDir:    tmp + "/config",
		RuntimeBaseDir:   tmp + "/runtime",
		ClaudeBinary:     "/opt/claude/bin/claude",
		WrapperBinary:    "/opt/ccgo/ccgo-wrapper",
		Args:             []string{"--debug"},
	})
	require.NoError(t, err)
	require.Equal(t, int64(123), spec.WorkspaceID)
	require.Equal(t, "/srv/ccgo/workspaces/u1/project", spec.Dir)
	require.Equal(t, "/opt/claude/bin/claude", spec.Binary)
	require.Equal(t, []string{"--append-system-prompt-file", spec.PromptFile, "--debug"}, spec.Args)
	require.FileExists(t, spec.PromptFile)
	require.Contains(t, envValue(spec.Env, "CLAUDE_CONFIG_DIR"), "/config/workspace-123")
	require.Equal(t, "/opt/ccgo/ccgo-wrapper", envValue(spec.Env, "CLAUDE_CODE_SHELL_PREFIX"))
	require.Equal(t, "123", envValue(spec.Env, shellwrapper.EnvWorkspaceID))
	require.Equal(t, "/srv/ccgo/workspaces/u1/project", envValue(spec.Env, shellwrapper.EnvServerRoot))
	require.Equal(t, "/Users/alice/project", envValue(spec.Env, shellwrapper.EnvLocalRoot))
	require.True(t, strings.HasSuffix(envValue(spec.Env, shellwrapper.EnvExecSocket), "/runtime/workspace-123/exec.sock"))
}

func TestBuildLaunchSpecRejectsMissingRoots(t *testing.T) {
	_, err := BuildLaunchSpec(LaunchConfig{WorkspaceID: 1, LocalRootDisplay: "/Users/alice/project"})
	require.ErrorContains(t, err, "server root")

	_, err = BuildLaunchSpec(LaunchConfig{WorkspaceID: 1, ServerRoot: "/srv/ccgo/project"})
	require.ErrorContains(t, err, "local root")
}

func TestBuildLaunchSpecDoesNotWritePromptIntoProjection(t *testing.T) {
	tmp := t.TempDir()
	serverRoot := tmp + "/workspace"
	require.NoError(t, os.Mkdir(serverRoot, 0o755))
	spec, err := BuildLaunchSpec(LaunchConfig{
		WorkspaceID:      1,
		ServerRoot:       serverRoot,
		LocalRootDisplay: "/Users/alice/project",
		ConfigBaseDir:    tmp + "/config",
		RuntimeBaseDir:   tmp + "/runtime",
	})
	require.NoError(t, err)
	require.NotContains(t, spec.PromptFile, serverRoot)
}

func envValue(env []string, key string) string {
	prefix := key + "="
	for _, item := range env {
		if strings.HasPrefix(item, prefix) {
			return strings.TrimPrefix(item, prefix)
		}
	}
	return ""
}
