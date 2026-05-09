package cli

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoginWithToken_SavesConfigAndRedactsToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")

	result, err := LoginWithToken(LoginOptions{
		Server:     "https://ccgo.test",
		User:       "alice",
		Token:      "ccgo_abcdefghijklmnopqrstuvwxyz",
		ConfigPath: path,
	})
	require.NoError(t, err)
	require.Equal(t, path, result.ConfigPath)
	require.Equal(t, "https://ccgo.test", result.Server)
	require.Equal(t, "ccgo...wxyz", result.RedactedToken)

	cfg, err := LoadConfig(path)
	require.NoError(t, err)
	require.Equal(t, "alice", cfg.User)
	require.Equal(t, "ccgo_abcdefghijklmnopqrstuvwxyz", cfg.Token)
	require.NotEmpty(t, cfg.DeviceID)
}

func TestLoginWithToken_RequiresToken(t *testing.T) {
	_, err := LoginWithToken(LoginOptions{Token: " "})
	require.ErrorContains(t, err, "token")
}
