package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	configDirName  = "ccgo"
	configFileName = "config.json"
)

type Config struct {
	Server   string `json:"server"`
	User     string `json:"user,omitempty"`
	Token    string `json:"token"`
	DeviceID string `json:"device_id"`
}

func DefaultConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(dir, configDirName, configFileName), nil
}

func LoadConfig(path string) (*Config, error) {
	if strings.TrimSpace(path) == "" {
		var err error
		path, err = DefaultConfigPath()
		if err != nil {
			return nil, err
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read ccgo config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse ccgo config: %w", err)
	}
	if strings.TrimSpace(cfg.Server) == "" || strings.TrimSpace(cfg.Token) == "" || strings.TrimSpace(cfg.DeviceID) == "" {
		return nil, fmt.Errorf("ccgo config is incomplete")
	}
	return &cfg, nil
}

func SaveConfig(path string, cfg Config) error {
	if strings.TrimSpace(path) == "" {
		var err error
		path, err = DefaultConfigPath()
		if err != nil {
			return err
		}
	}
	if strings.TrimSpace(cfg.Server) == "" || strings.TrimSpace(cfg.Token) == "" || strings.TrimSpace(cfg.DeviceID) == "" {
		return fmt.Errorf("ccgo config is incomplete")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create ccgo config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode ccgo config: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write ccgo config: %w", err)
	}
	return nil
}

func RedactToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "..." + token[len(token)-4:]
}
