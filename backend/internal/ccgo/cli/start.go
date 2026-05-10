package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/agent"
)

const (
	defaultAgentReadyTimeout      = 10 * time.Second
	defaultAgentReadyPollInterval = 250 * time.Millisecond
)

type StartOptions struct {
	LocalPath              string
	ConfigPath             string
	NoAgent                bool
	AgentReadyTimeout      time.Duration
	AgentReadyPollInterval time.Duration
	Preflight              func(context.Context, string) (*LocalPreflightResult, error)
	Resolver               interface {
		Resolve(context.Context, string) (*WorkspaceResolution, error)
		StartWorkstation(context.Context, int64) (*StartWorkstationResult, error)
	}
	Connector func(context.Context, agent.ConnectorOptions) error
}

type StartResult struct {
	WorkspaceID       int64
	LocalRoot         string
	LocalRootRedacted string
	AgentConnected    bool
	RunID             string
	RunStatus         string
	ReusedRun         bool
}

type startResolver interface {
	Resolve(context.Context, string) (*WorkspaceResolution, error)
	StartWorkstation(context.Context, int64) (*StartWorkstationResult, error)
}

type agentStatusResolver interface {
	WorkstationStatus(context.Context, int64) (*WorkstationStatusResult, error)
}

func Start(ctx context.Context, opts StartOptions) (*StartResult, error) {
	localPath := strings.TrimSpace(opts.LocalPath)
	if localPath == "" {
		return nil, fmt.Errorf("local path is required")
	}
	if !opts.NoAgent {
		preflight := opts.Preflight
		if preflight == nil {
			preflight = CheckLocalPreflight
		}
		if _, err := preflight(ctx, localPath); err != nil {
			return nil, err
		}
	}
	resolver := opts.Resolver
	if resolver == nil {
		resolver = WorkspaceClient{ConfigPath: opts.ConfigPath}
	}
	resolution, err := resolver.Resolve(ctx, localPath)
	if err != nil {
		return nil, err
	}
	if err := rememberLastWorkspace(opts.ConfigPath, resolver, resolution.Workspace.ID); err != nil {
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
	connectorErr := make(chan error, 1)
	go func() {
		connectorErr <- connector(ctx, connectorOptions(resolution))
	}()
	if err := waitForAgentReady(ctx, resolver, resolution.Workspace.ID, connectorErr, opts.AgentReadyTimeout, opts.AgentReadyPollInterval); err != nil {
		return nil, err
	}
	result.AgentConnected = true
	started, err := resolver.StartWorkstation(ctx, resolution.Workspace.ID)
	if err != nil {
		return nil, err
	}
	result.RunID = started.Run.RunID
	result.RunStatus = started.Run.Status
	result.ReusedRun = started.Reused
	return result, nil
}

func waitForAgentReady(
	ctx context.Context,
	resolver startResolver,
	workspaceID int64,
	connectorErr <-chan error,
	timeout time.Duration,
	pollInterval time.Duration,
) error {
	if timeout <= 0 {
		timeout = defaultAgentReadyTimeout
	}
	if pollInterval <= 0 {
		pollInterval = defaultAgentReadyPollInterval
	}
	statusResolver, hasStatus := resolver.(agentStatusResolver)
	if !hasStatus {
		select {
		case err := <-connectorErr:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	readyCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		status, err := statusResolver.WorkstationStatus(readyCtx, workspaceID)
		if err == nil && status != nil && status.AgentConnected {
			return nil
		}
		select {
		case err := <-connectorErr:
			if err != nil {
				return err
			}
			return fmt.Errorf("ccgo local agent connection ended before the workstation was ready")
		case <-readyCtx.Done():
			return fmt.Errorf("ccgo local agent did not become ready: %w", readyCtx.Err())
		case <-ticker.C:
		}
	}
}

func connectorOptions(resolution *WorkspaceResolution) agent.ConnectorOptions {
	if resolution == nil || resolution.Config == nil || resolution.LocalPath == nil {
		return agent.ConnectorOptions{}
	}
	return agent.ConnectorOptions{
		Server: resolution.Config.Server,
		Token:  resolution.AgentCredential.Token,
		Nonce:  resolution.AgentCredential.Nonce,
		Root:   resolution.LocalPath.CanonicalRoot,
	}
}

func rememberLastWorkspace(configPath string, resolver startResolver, workspaceID int64) error {
	if workspaceID <= 0 {
		return nil
	}
	client, ok := resolver.(WorkspaceClient)
	if !ok {
		return nil
	}
	cfg := client.Config
	if cfg == nil {
		var err error
		cfg, err = LoadConfig(configPath)
		if err != nil {
			return err
		}
	}
	cfg.LastWorkspaceID = workspaceID
	return SaveConfig(configPath, *cfg)
}
