package cli

import "context"

type workstationStopClient interface {
	StopWorkstation(context.Context, int64, string) (*StopWorkstationResult, error)
}

type StopOptions struct {
	ConfigPath  string
	Config      *Config
	WorkspaceID int64
	Reason      string
	Client      workstationStopClient
}

func Stop(ctx context.Context, opts StopOptions) (*StopWorkstationResult, error) {
	workspaceID, cfg, err := lifecycleWorkspaceID(opts.ConfigPath, opts.Config, opts.WorkspaceID)
	if err != nil {
		return nil, err
	}
	client := opts.Client
	if client == nil {
		client = WorkspaceClient{ConfigPath: opts.ConfigPath, Config: cfg}
	}
	return client.StopWorkstation(ctx, workspaceID, opts.Reason)
}
