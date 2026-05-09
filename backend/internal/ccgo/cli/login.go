package cli

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

const defaultServerURL = "https://ccgo.example.com"

type LoginOptions struct {
	Server     string
	Token      string
	User       string
	ConfigPath string
}

type LoginResult struct {
	ConfigPath    string
	Server        string
	User          string
	RedactedToken string
}

func LoginWithToken(opts LoginOptions) (*LoginResult, error) {
	token := strings.TrimSpace(opts.Token)
	if token == "" {
		return nil, fmt.Errorf("token is required")
	}
	server := strings.TrimSpace(opts.Server)
	if server == "" {
		server = defaultServerURL
	}
	deviceID, err := randomDeviceID()
	if err != nil {
		return nil, err
	}
	cfg := Config{
		Server:   server,
		User:     strings.TrimSpace(opts.User),
		Token:    token,
		DeviceID: deviceID,
	}
	path := opts.ConfigPath
	if path == "" {
		path, err = DefaultConfigPath()
		if err != nil {
			return nil, err
		}
	}
	if err := SaveConfig(path, cfg); err != nil {
		return nil, err
	}
	return &LoginResult{
		ConfigPath:    path,
		Server:        server,
		User:          cfg.User,
		RedactedToken: RedactToken(token),
	}, nil
}

func randomDeviceID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate ccgo device id: %w", err)
	}
	return "dev_" + hex.EncodeToString(buf), nil
}
