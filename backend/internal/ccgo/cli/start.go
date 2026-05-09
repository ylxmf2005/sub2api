package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/agent"
)

type StartOptions struct {
	LocalPath  string
	ConfigPath string
	NoAgent    bool
	Resolver   interface {
		Resolve(context.Context, string) (*WorkspaceResolution, error)
	}
	Connector func(context.Context, agent.ConnectorOptions) error
}

type StartResult struct {
	WorkspaceID       int64
	LocalRoot         string
	LocalRootRedacted string
	AgentConnected    bool
}

func Start(ctx context.Context, opts StartOptions) (*StartResult, error) {
	localPath := strings.TrimSpace(opts.LocalPath)
	if localPath == "" {
		return nil, fmt.Errorf("local path is required")
	}
	resolver := opts.Resolver
	if resolver == nil {
		resolver = WorkspaceClient{ConfigPath: opts.ConfigPath}
	}
	resolution, err := resolver.Resolve(ctx, localPath)
	if err != nil {
		return nil, err
	}
	result := &StartResult{
		WorkspaceID:       resolution.Workspace.ID,
		LocalRoot:         resolution.LocalPath.LocalRootDisplay,
		LocalRootRedacted: resolution.Workspace.LocalRootRedacted,
	}
	if opts.NoAgent {
		return result, nil
	}
	connector := opts.Connector
	if connector == nil {
		connector = agent.RunConnector
	}
	if err := connector(ctx, agent.ConnectorOptions{
		Server: resolution.Config.Server,
		Token:  resolution.AgentCredential.Token,
		Nonce:  resolution.AgentCredential.Nonce,
		Root:   resolution.LocalPath.CanonicalRoot,
	}); err != nil {
		return nil, err
	}
	result.AgentConnected = true
	return result, nil
}
