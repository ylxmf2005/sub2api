package cli

import (
	"context"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/agent"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}

type resolverFunc func(context.Context, string) (*WorkspaceResolution, error)

func (f resolverFunc) Resolve(ctx context.Context, localPath string) (*WorkspaceResolution, error) {
	return f(ctx, localPath)
}

func (f resolverFunc) StartWorkstation(context.Context, int64) (*StartWorkstationResult, error) {
	return &StartWorkstationResult{
		Run: WorkstationRun{
			WorkspaceID: 42,
			RunID:       "run_test",
			Status:      "running",
		},
	}, nil
}

type startResolverStub struct {
	resolution       *WorkspaceResolution
	statusCalls      int
	startedWorkspace int64
}

func (s *startResolverStub) Resolve(context.Context, string) (*WorkspaceResolution, error) {
	return s.resolution, nil
}

func (s *startResolverStub) StartWorkstation(_ context.Context, workspaceID int64) (*StartWorkstationResult, error) {
	s.startedWorkspace = workspaceID
	return &StartWorkstationResult{
		Run: WorkstationRun{
			WorkspaceID: workspaceID,
			RunID:       "run_test",
			Status:      "running",
		},
	}, nil
}

func (s *startResolverStub) WorkstationStatus(context.Context, int64) (*WorkstationStatusResult, error) {
	s.statusCalls++
	return &WorkstationStatusResult{Workspace: Workspace{ID: s.resolution.Workspace.ID}, AgentConnected: s.statusCalls >= 2}, nil
}

func TestStartWithWorkspaceClientPersistsLastWorkspace(t *testing.T) {
	project := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, SaveConfig(configPath, Config{
		Server:   "https://ccgo.example.com",
		Token:    "token",
		DeviceID: "dev_123",
	}))
	serverURL := "https://ccgo.example.com"
	result, err := Start(context.Background(), StartOptions{
		LocalPath:  project,
		NoAgent:    true,
		ConfigPath: configPath,
		Resolver: WorkspaceClient{
			ConfigPath: configPath,
			Config: &Config{
				Server:   serverURL,
				Token:    "token",
				DeviceID: "dev_123",
			},
			HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				require.Equal(t, workspaceResolvePath, req.URL.Path)
				return jsonResponse(`{
					"code": 0,
					"message": "success",
					"data": {
						"workspace": {
							"id": 42,
							"server_root": "/var/lib/ccgo/workspaces/user-1/u1-abcd",
							"local_root_redacted": ".../project"
						},
						"agent_credential": {
							"token": "agent_token",
							"nonce": "agent_nonce",
							"expires_at": "2026-05-09T10:00:00Z"
						}
					}
				}`), nil
			})},
		},
	})
	require.NoError(t, err)
	require.Equal(t, int64(42), result.WorkspaceID)
	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.Equal(t, int64(42), cfg.LastWorkspaceID)
}

func TestStartResolvesWorkspaceAndConnectsAgent(t *testing.T) {
	var connected agent.ConnectorOptions
	resolver := &startResolverStub{resolution: &WorkspaceResolution{
		Workspace: Workspace{
			ID:                42,
			LocalRootRedacted: ".../project",
		},
		AgentCredential: AgentCredential{
			Token:     "agent_token",
			Nonce:     "agent_nonce",
			ExpiresAt: time.Now().Add(time.Minute),
		},
		LocalPath: &LocalPathInfo{
			CanonicalRoot:    "/Users/alice/project",
			LocalRootDisplay: "/Users/alice/project",
		},
		Config: &Config{Server: "https://ccgo.example.com"},
	}}
	connectorReady := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result, err := Start(ctx, StartOptions{
		LocalPath:              "/Users/alice/project",
		Resolver:               resolver,
		AgentReadyTimeout:      time.Second,
		AgentReadyPollInterval: time.Millisecond,
		Preflight: func(context.Context, string) (*LocalPreflightResult, error) {
			return &LocalPreflightResult{}, nil
		},
		Connector: func(ctx context.Context, opts agent.ConnectorOptions) error {
			connected = opts
			close(connectorReady)
			<-ctx.Done()
			return ctx.Err()
		},
	})
	require.NoError(t, err)
	<-connectorReady
	require.Equal(t, int64(42), result.WorkspaceID)
	require.Equal(t, "/Users/alice/project", result.LocalRoot)
	require.Equal(t, ".../project", result.LocalRootRedacted)
	require.True(t, result.AgentConnected)
	require.Equal(t, "run_test", result.RunID)
	require.Equal(t, "running", result.RunStatus)
	require.Equal(t, "https://ccgo.example.com", connected.Server)
	require.Equal(t, "agent_token", connected.Token)
	require.Equal(t, "agent_nonce", connected.Nonce)
	require.Equal(t, "/Users/alice/project", connected.Root)
	require.Equal(t, int64(42), resolver.startedWorkspace)
}

func TestStartNoAgentOnlyResolvesWorkspace(t *testing.T) {
	calledConnector := false
	result, err := Start(context.Background(), StartOptions{
		LocalPath: "/Users/alice/project",
		NoAgent:   true,
		Resolver: resolverFunc(func(context.Context, string) (*WorkspaceResolution, error) {
			return &WorkspaceResolution{
				Workspace: Workspace{ID: 42},
				LocalPath: &LocalPathInfo{
					CanonicalRoot:    "/Users/alice/project",
					LocalRootDisplay: "/Users/alice/project",
				},
				Config: &Config{Server: "https://ccgo.example.com"},
			}, nil
		}),
		Connector: func(context.Context, agent.ConnectorOptions) error {
			calledConnector = true
			return nil
		},
	})
	require.NoError(t, err)
	require.Equal(t, int64(42), result.WorkspaceID)
	require.False(t, result.AgentConnected)
	require.False(t, calledConnector)
}

func TestStartRunsLocalPreflightBeforeResolvingWorkspace(t *testing.T) {
	resolverCalled := false
	_, err := Start(context.Background(), StartOptions{
		LocalPath: "/Users/alice/project",
		Resolver: resolverFunc(func(context.Context, string) (*WorkspaceResolution, error) {
			resolverCalled = true
			return nil, errors.New("resolver should not run")
		}),
		Preflight: func(context.Context, string) (*LocalPreflightResult, error) {
			return nil, errors.New("preflight failed")
		},
	})
	require.ErrorContains(t, err, "preflight failed")
	require.False(t, resolverCalled)
}
