package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestLoginDeviceFlowStoresApprovedToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	var gotStart map[string]string
	var pollCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case deviceLoginStartPath:
			require.NoError(t, json.NewDecoder(r.Body).Decode(&gotStart))
			_, _ = w.Write([]byte(`{
				"code": 0,
				"message": "success",
				"data": {
					"device_code": "device_code_123",
					"user_code": "ABCD-EFGH",
					"verification_uri": "https://ccgo.test/ccgo/device?code=ABCD-EFGH",
					"expires_at": "2026-05-09T10:00:00Z",
					"interval_seconds": 1
				}
			}`))
		case deviceLoginPollPath:
			pollCount++
			if pollCount == 1 {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"message":"ccgo device login is pending approval","reason":"CCGO_DEVICE_LOGIN_PENDING"}`))
				return
			}
			_, _ = w.Write([]byte(`{
				"code": 0,
				"message": "success",
				"data": {
					"status": "approved",
					"access_token": "access_123",
					"refresh_token": "refresh_123",
					"expires_in": 3600,
					"token_type": "Bearer"
				}
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	var opened string
	var output strings.Builder
	result, err := Login(context.Background(), LoginOptions{
		Server:       server.URL,
		ConfigPath:   path,
		DeviceID:     "dev_test",
		Output:       &output,
		OpenBrowser:  func(url string) error { opened = url; return nil },
		PollInterval: time.Millisecond,
		PollTimeout:  time.Second,
	})
	require.NoError(t, err)
	require.Equal(t, server.URL, result.Server)
	require.Equal(t, "acce..._123", result.RedactedToken)
	require.Equal(t, "dev_test", gotStart["device_id"])
	require.Equal(t, "https://ccgo.test/ccgo/device?code=ABCD-EFGH", opened)
	require.Contains(t, output.String(), "ABCD-EFGH")
	require.Equal(t, 2, pollCount)

	cfg, err := LoadConfig(path)
	require.NoError(t, err)
	require.Equal(t, "access_123", cfg.Token)
	require.Equal(t, "dev_test", cfg.DeviceID)
}

func TestLoginDeviceFlowSurfacesExpiredDeviceCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case deviceLoginStartPath:
			_, _ = w.Write([]byte(`{
				"code": 0,
				"message": "success",
				"data": {
					"device_code": "device_code_123",
					"user_code": "ABCD-EFGH",
					"verification_uri": "https://ccgo.test/ccgo/device?code=ABCD-EFGH",
					"expires_at": "2026-05-09T10:00:00Z",
					"interval_seconds": 1
				}
			}`))
		case deviceLoginPollPath:
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"message":"ccgo device login has expired","reason":"CCGO_DEVICE_LOGIN_EXPIRED"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	_, err := Login(context.Background(), LoginOptions{
		Server:       server.URL,
		ConfigPath:   filepath.Join(t.TempDir(), "config.json"),
		DeviceID:     "dev_test",
		OpenBrowser:  func(string) error { return nil },
		PollInterval: time.Millisecond,
		PollTimeout:  time.Second,
	})
	require.ErrorContains(t, err, "CCGO_DEVICE_LOGIN_EXPIRED")
}
