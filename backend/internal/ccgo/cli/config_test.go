package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSaveLoadConfig_UsesPrivateConfigFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ccgo", "config.json")

	require.NoError(t, SaveConfig(path, Config{
		Server:   "https://ccgo.test",
		User:     "alice",
		Token:    "ccgo_1234567890",
		DeviceID: "dev_abc",
	}))

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	cfg, err := LoadConfig(path)
	require.NoError(t, err)
	require.Equal(t, "https://ccgo.test", cfg.Server)
	require.Equal(t, "alice", cfg.User)
	require.Equal(t, "ccgo_1234567890", cfg.Token)
	require.Equal(t, "dev_abc", cfg.DeviceID)
}

func TestRedactToken(t *testing.T) {
	require.Equal(t, "ccgo...7890", RedactToken("ccgo_1234567890"))
	require.Equal(t, "****", RedactToken("short"))
	require.Equal(t, "", RedactToken(""))
}
