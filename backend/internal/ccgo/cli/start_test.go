package cli

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/agent"
	"github.com/stretchr/testify/require"
)

type resolverFunc func(context.Context, string) (*WorkspaceResolution, error)

func (f resolverFunc) Resolve(ctx context.Context, localPath string) (*WorkspaceResolution, error) {
	return f(ctx, localPath)
}

func TestStartResolvesWorkspaceAndConnectsAgent(t *testing.T) {
	var connected agent.ConnectorOptions
	result, err := Start(context.Background(), StartOptions{
		LocalPath: "/Users/alice/project",
		Resolver: resolverFunc(func(ctx context.Context, localPath string) (*WorkspaceResolution, error) {
			require.Equal(t, "/Users/alice/project", localPath)
			return &WorkspaceResolution{
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
			}, nil
		}),
		Connector: func(ctx context.Context, opts agent.ConnectorOptions) error {
			connected = opts
			return nil
		},
	})
	require.NoError(t, err)
	require.Equal(t, int64(42), result.WorkspaceID)
	require.Equal(t, "/Users/alice/project", result.LocalRoot)
	require.Equal(t, ".../project", result.LocalRootRedacted)
	require.True(t, result.AgentConnected)
	require.Equal(t, "https://ccgo.example.com", connected.Server)
	require.Equal(t, "agent_token", connected.Token)
	require.Equal(t, "agent_nonce", connected.Nonce)
	require.Equal(t, "/Users/alice/project", connected.Root)
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
