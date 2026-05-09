package cli

import (
	"context"
	"fmt"
)

type workstationStatusClient interface {
	WorkstationStatus(context.Context, int64) (*WorkstationStatusResult, error)
}

type StatusOptions struct {
	ConfigPath  string
	Config      *Config
	WorkspaceID int64
	Client      workstationStatusClient
}

func Status(ctx context.Context, opts StatusOptions) (*WorkstationStatusResult, error) {
	workspaceID, cfg, err := lifecycleWorkspaceID(opts.ConfigPath, opts.Config, opts.WorkspaceID)
	if err != nil {
		return nil, err
	}
	client := opts.Client
	if client == nil {
		client = WorkspaceClient{ConfigPath: opts.ConfigPath, Config: cfg}
	}
	return client.WorkstationStatus(ctx, workspaceID)
}

func lifecycleWorkspaceID(configPath string, cfg *Config, explicit int64) (int64, *Config, error) {
	if explicit > 0 {
		return explicit, cfg, nil
	}
	var err error
	if cfg == nil {
		cfg, err = LoadConfig(configPath)
		if err != nil {
			return 0, nil, err
		}
	}
	if cfg.LastWorkspaceID <= 0 {
		return 0, nil, fmt.Errorf("workspace id is required; run ccgo <local_path> first or pass a workspace id")
	}
	return cfg.LastWorkspaceID, cfg, nil
}
