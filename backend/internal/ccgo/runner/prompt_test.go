package runner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGeneratePromptMakesLocalRootCanonical(t *testing.T) {
	prompt, err := GeneratePrompt(PromptConfig{
		LocalRootDisplay: "/Users/alice/My Project",
		ServerRoot:       "/srv/ccgo/workspaces/u1/project",
		PathStyle:        "posix",
	})
	require.NoError(t, err)
	require.Contains(t, prompt, "Canonical local workspace: /Users/alice/My Project")
	require.Contains(t, prompt, "Internal server projection: /srv/ccgo/workspaces/u1/project")
	require.Contains(t, prompt, "Shell commands launched through Claude Code execute on the user's local machine")
	require.Contains(t, prompt, "prefer project-relative paths")
	require.Contains(t, prompt, "the local path is the source of truth")
}

func TestWritePromptFileOutsideWorkspace(t *testing.T) {
	tmp := t.TempDir()
	workspace := filepath.Join(tmp, "workspace")
	runtimeDir := filepath.Join(tmp, "runtime")
	require.NoError(t, os.Mkdir(workspace, 0o755))

	path, err := WritePromptFile(runtimeDir, PromptConfig{
		LocalRootDisplay: "/Users/alice/project",
		ServerRoot:       workspace,
		PathStyle:        "posix",
	})
	require.NoError(t, err)
	require.NotContains(t, path, workspace)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(data), "ccgo Reverse Workstation")
}

func TestGeneratePromptRejectsAmbiguousRoots(t *testing.T) {
	_, err := GeneratePrompt(PromptConfig{ServerRoot: "/srv/ccgo/project"})
	require.ErrorContains(t, err, "local root")

	_, err = GeneratePrompt(PromptConfig{LocalRootDisplay: "/Users/alice/project"})
	require.ErrorContains(t, err, "server root")
}
