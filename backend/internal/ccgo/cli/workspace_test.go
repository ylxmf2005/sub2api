package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWorkspaceClientResolvePostsCanonicalPathMetadata(t *testing.T) {
	project := filepath.Join(t.TempDir(), "project")
	require.NoError(t, os.Mkdir(project, 0o755))
	canonicalProject, err := filepath.EvalSymlinks(project)
	require.NoError(t, err)

	var gotAuth string
	var gotPayload map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, workspaceResolvePath, r.URL.Path)
		gotAuth = r.Header.Get("Authorization")
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotPayload))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"code": 0,
			"message": "success",
			"data": {
				"workspace": {
					"id": 42,
					"workspace_slug": "u1-abcd",
					"server_root": "/var/lib/ccgo/workspaces/user-1/u1-abcd",
					"local_root_display": "` + jsonEscape(project) + `",
					"local_root_redacted": ".../project",
					"os": "darwin",
					"path_style": "posix",
					"status": "active"
				},
				"agent_credential": {
					"token": "agent_token",
					"nonce": "agent_nonce",
					"expires_at": "2026-05-09T10:00:00Z"
				},
				"created": true
			}
		}`))
	}))
	defer server.Close()

	resolution, err := WorkspaceClient{Config: &Config{
		Server:   server.URL,
		Token:    "ccgo_cli_token",
		DeviceID: "dev_123",
	}}.Resolve(context.Background(), project)
	require.NoError(t, err)

	require.Equal(t, "Bearer ccgo_cli_token", gotAuth)
	require.Equal(t, "dev_123", gotPayload["device_id"])
	require.Equal(t, canonicalProject, gotPayload["local_root_display"])
	require.Len(t, gotPayload["local_root_hash"], 64)
	require.Equal(t, int64(42), resolution.Workspace.ID)
	require.Equal(t, "/var/lib/ccgo/workspaces/user-1/u1-abcd", resolution.Workspace.ServerRoot)
	require.Equal(t, "agent_token", resolution.AgentCredential.Token)
	require.True(t, resolution.Created)
	require.Equal(t, canonicalProject, resolution.LocalPath.LocalRootDisplay)
	require.Equal(t, time.Date(2026, 5, 9, 10, 0, 0, 0, time.UTC), resolution.AgentCredential.ExpiresAt)
}

func TestWorkspaceClientResolveRequiresAgentCredential(t *testing.T) {
	project := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"code": 0,
			"message": "success",
			"data": {
				"workspace": {
					"id": 42,
					"server_root": "/var/lib/ccgo/workspaces/user-1/u1-abcd"
				},
				"agent_credential": {}
			}
		}`))
	}))
	defer server.Close()

	_, err := WorkspaceClient{Config: &Config{
		Server:   server.URL,
		Token:    "ccgo_cli_token",
		DeviceID: "dev_123",
	}}.Resolve(context.Background(), project)
	require.ErrorContains(t, err, "agent credentials")
}

func TestWorkspaceClientResolveSurfacesAPIError(t *testing.T) {
	project := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"message":"ccgo workspace is already bound to another device","reason":"CCGO_DEVICE_MISMATCH"}`))
	}))
	defer server.Close()

	_, err := WorkspaceClient{Config: &Config{
		Server:   server.URL,
		Token:    "ccgo_cli_token",
		DeviceID: "dev_123",
	}}.Resolve(context.Background(), project)
	require.ErrorContains(t, err, "CCGO_DEVICE_MISMATCH")
}

func jsonEscape(value string) string {
	data, _ := json.Marshal(value)
	return string(data[1 : len(data)-1])
}
