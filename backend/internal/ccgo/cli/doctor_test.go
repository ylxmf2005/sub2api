package cli

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDoctorReportsConfigAndOptionalLocalPreflight(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, SaveConfig(configPath, Config{
		Server:   "https://ccgo.test",
		User:     "alice",
		Token:    "ccgo_1234567890",
		DeviceID: "dev_123",
	}))

	result, err := Doctor(context.Background(), DoctorOptions{
		ConfigPath: configPath,
		LocalPath:  "/project",
		Preflight: func(ctx context.Context, localPath string) (*LocalPreflightResult, error) {
			require.Equal(t, "/project", localPath)
			return &LocalPreflightResult{
				LocalRoot:  "/project",
				OS:         "darwin",
				PathStyle:  "posix",
				ShellName:  "bash",
				ShellPath:  "/bin/bash",
				ProbeValue: localPreflightProbe,
			}, nil
		},
	})
	require.NoError(t, err)
	require.Equal(t, configPath, result.ConfigPath)
	require.Equal(t, "https://ccgo.test", result.Server)
	require.Equal(t, "alice", result.User)
	require.Equal(t, "dev_123", result.DeviceID)
	require.Equal(t, "ccgo...7890", result.RedactedToken)
	require.NotNil(t, result.LocalPreflight)
	require.Equal(t, "/project", result.LocalPreflight.LocalRoot)
}

func TestDoctorWithoutLocalPathOnlyChecksConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, SaveConfig(configPath, Config{
		Server:   "https://ccgo.test",
		Token:    "ccgo_1234567890",
		DeviceID: "dev_123",
	}))

	calledPreflight := false
	result, err := Doctor(context.Background(), DoctorOptions{
		ConfigPath: configPath,
		Preflight: func(context.Context, string) (*LocalPreflightResult, error) {
			calledPreflight = true
			return nil, nil
		},
	})
	require.NoError(t, err)
	require.Nil(t, result.LocalPreflight)
	require.False(t, calledPreflight)
}

func TestDoctorSurfacesLocalPreflightFailure(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, SaveConfig(configPath, Config{
		Server:   "https://ccgo.test",
		Token:    "ccgo_1234567890",
		DeviceID: "dev_123",
	}))

	_, err := Doctor(context.Background(), DoctorOptions{
		ConfigPath: configPath,
		LocalPath:  "/project",
		Preflight: func(context.Context, string) (*LocalPreflightResult, error) {
			return nil, errors.New("bash missing")
		},
	})
	require.ErrorContains(t, err, "local preflight check failed")
	require.ErrorContains(t, err, "bash missing")
}
