package cli

import (
	"context"
	"fmt"
	"strings"
)

type DoctorOptions struct {
	LocalPath  string
	ConfigPath string
	Preflight  func(context.Context, string) (*LocalPreflightResult, error)
}

type DoctorResult struct {
	ConfigPath     string                `json:"config_path"`
	Server         string                `json:"server"`
	User           string                `json:"user,omitempty"`
	DeviceID       string                `json:"device_id"`
	RedactedToken  string                `json:"redacted_token"`
	LocalPreflight *LocalPreflightResult `json:"local_preflight,omitempty"`
}

func Doctor(ctx context.Context, opts DoctorOptions) (*DoctorResult, error) {
	cfg, err := LoadConfig(opts.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("ccgo doctor: login config check failed: %w", err)
	}
	configPath := strings.TrimSpace(opts.ConfigPath)
	if configPath == "" {
		configPath, err = DefaultConfigPath()
		if err != nil {
			return nil, fmt.Errorf("ccgo doctor: resolve config path: %w", err)
		}
	}
	result := &DoctorResult{
		ConfigPath:    configPath,
		Server:        cfg.Server,
		User:          cfg.User,
		DeviceID:      cfg.DeviceID,
		RedactedToken: RedactToken(cfg.Token),
	}
	localPath := strings.TrimSpace(opts.LocalPath)
	if localPath == "" {
		return result, nil
	}
	preflight := opts.Preflight
	if preflight == nil {
		preflight = CheckLocalPreflight
	}
	local, err := preflight(ctx, localPath)
	if err != nil {
		return nil, fmt.Errorf("ccgo doctor: local preflight check failed: %w", err)
	}
	result.LocalPreflight = local
	return result, nil
}
