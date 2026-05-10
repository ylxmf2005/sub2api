package cli

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckLocalPreflightRequiresRunnableBash(t *testing.T) {
	result, err := checkLocalPreflight(context.Background(), "/project", localPreflightOptions{
		goos: "darwin",
		resolvePath: func(string) (*LocalPathInfo, error) {
			return &LocalPathInfo{
				CanonicalRoot: "/project",
				OS:            "darwin",
				PathStyle:     "posix",
			}, nil
		},
		lookPath: func(name string) (string, error) {
			require.Equal(t, "bash", name)
			return "/bin/bash", nil
		},
		runShellProbe: func(ctx context.Context, shellPath string, localRoot string) (string, error) {
			require.Equal(t, "/bin/bash", shellPath)
			require.Equal(t, "/project", localRoot)
			return localPreflightProbe, nil
		},
	})
	require.NoError(t, err)
	require.Equal(t, "/project", result.LocalRoot)
	require.Equal(t, "bash", result.ShellName)
	require.Equal(t, "/bin/bash", result.ShellPath)
}

func TestCheckLocalPreflightWindowsMissingBashExplainsWSLOrGitBash(t *testing.T) {
	_, err := checkLocalPreflight(context.Background(), `C:\Users\Alice\project`, localPreflightOptions{
		goos: "windows",
		resolvePath: func(string) (*LocalPathInfo, error) {
			return &LocalPathInfo{
				CanonicalRoot: `C:\Users\Alice\project`,
				OS:            "windows",
				PathStyle:     "windows",
			}, nil
		},
		lookPath: func(string) (string, error) {
			return "", errors.New("not found")
		},
	})
	require.ErrorContains(t, err, "Git Bash")
	require.ErrorContains(t, err, "WSL")
}

func TestCheckLocalPreflightRejectsUnexpectedProbeOutput(t *testing.T) {
	_, err := checkLocalPreflight(context.Background(), "/project", localPreflightOptions{
		goos: "linux",
		resolvePath: func(string) (*LocalPathInfo, error) {
			return &LocalPathInfo{
				CanonicalRoot: "/project",
				OS:            "linux",
				PathStyle:     "posix",
			}, nil
		},
		lookPath: func(string) (string, error) {
			return "/bin/bash", nil
		},
		runShellProbe: func(context.Context, string, string) (string, error) {
			return "wrong", nil
		},
	})
	require.ErrorContains(t, err, "bash probe")
}
