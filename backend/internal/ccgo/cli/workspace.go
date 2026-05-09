package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const workspaceResolvePath = "/api/v1/ccgo/workspaces/resolve"
const workstationStartPath = "/api/v1/ccgo/workstations/start"

type Workspace struct {
	ID                int64  `json:"id"`
	WorkspaceSlug     string `json:"workspace_slug"`
	ServerRoot        string `json:"server_root"`
	LocalRootDisplay  string `json:"local_root_display"`
	LocalRootRedacted string `json:"local_root_redacted"`
	OS                string `json:"os"`
	PathStyle         string `json:"path_style"`
	Status            string `json:"status"`
}

type AgentCredential struct {
	Token     string    `json:"token"`
	Nonce     string    `json:"nonce"`
	ExpiresAt time.Time `json:"expires_at"`
}

type WorkspaceResolution struct {
	Workspace       Workspace       `json:"workspace"`
	AgentCredential AgentCredential `json:"agent_credential"`
	Created         bool            `json:"created"`
	DeviceConflict  *DeviceConflict `json:"device_conflict,omitempty"`
	LocalPath       *LocalPathInfo  `json:"-"`
	Config          *Config         `json:"-"`
}

type DeviceConflict struct {
	ExistingDeviceID  string `json:"existing_device_id"`
	RequestedDeviceID string `json:"requested_device_id"`
}

type WorkspaceClient struct {
	HTTPClient *http.Client
	ConfigPath string
	Config     *Config
}

func (c WorkspaceClient) Resolve(ctx context.Context, localPath string) (*WorkspaceResolution, error) {
	info, err := ResolveLocalPath(localPath)
	if err != nil {
		return nil, err
	}
	cfg := c.Config
	if cfg == nil {
		cfg, err = LoadConfig(c.ConfigPath)
		if err != nil {
			return nil, err
		}
	}
	payload := map[string]string{
		"canonical_root":     info.CanonicalRoot,
		"local_root_display": info.LocalRootDisplay,
		"local_root_hash":    info.LocalRootHash,
		"os":                 info.OS,
		"path_style":         info.PathStyle,
		"device_id":          cfg.DeviceID,
	}
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(payload); err != nil {
		return nil, fmt.Errorf("encode workspace request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, joinURL(cfg.Server, workspaceResolvePath), &body)
	if err != nil {
		return nil, fmt.Errorf("build workspace request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("resolve ccgo workspace: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("read workspace response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, decodeAPIError(resp.StatusCode, data)
	}
	var envelope struct {
		Code    int                 `json:"code"`
		Message string              `json:"message"`
		Data    WorkspaceResolution `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("parse workspace response: %w", err)
	}
	if envelope.Code != 0 {
		return nil, fmt.Errorf("resolve ccgo workspace: %s", envelope.Message)
	}
	result := envelope.Data
	result.LocalPath = info
	result.Config = cfg
	if result.Workspace.ID <= 0 || strings.TrimSpace(result.Workspace.ServerRoot) == "" {
		return nil, fmt.Errorf("workspace response is incomplete")
	}
	if strings.TrimSpace(result.AgentCredential.Token) == "" || strings.TrimSpace(result.AgentCredential.Nonce) == "" {
		return nil, fmt.Errorf("workspace response is missing agent credentials")
	}
	return &result, nil
}

type WorkstationRun struct {
	WorkspaceID int64  `json:"workspace_id"`
	RunID       string `json:"run_id"`
	Status      string `json:"status"`
}

type StartWorkstationResult struct {
	Workspace Workspace      `json:"workspace"`
	Run       WorkstationRun `json:"run"`
	Reused    bool           `json:"reused"`
}

func (c WorkspaceClient) StartWorkstation(ctx context.Context, workspaceID int64) (*StartWorkstationResult, error) {
	if workspaceID <= 0 {
		return nil, fmt.Errorf("workspace id is required")
	}
	cfg := c.Config
	var err error
	if cfg == nil {
		cfg, err = LoadConfig(c.ConfigPath)
		if err != nil {
			return nil, err
		}
	}
	payload := map[string]int64{"workspace_id": workspaceID}
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(payload); err != nil {
		return nil, fmt.Errorf("encode workstation start request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, joinURL(cfg.Server, workstationStartPath), &body)
	if err != nil {
		return nil, fmt.Errorf("build workstation start request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("start ccgo workstation: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("read workstation start response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, decodeAPIError(resp.StatusCode, data)
	}
	var envelope struct {
		Code    int                    `json:"code"`
		Message string                 `json:"message"`
		Data    StartWorkstationResult `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("parse workstation start response: %w", err)
	}
	if envelope.Code != 0 {
		return nil, fmt.Errorf("start ccgo workstation: %s", envelope.Message)
	}
	if strings.TrimSpace(envelope.Data.Run.RunID) == "" || strings.TrimSpace(envelope.Data.Run.Status) == "" {
		return nil, fmt.Errorf("workstation start response is incomplete")
	}
	return &envelope.Data, nil
}

func joinURL(base, path string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		base = defaultServerURL
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return base + path
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + path
	return parsed.String()
}

func decodeAPIError(statusCode int, data []byte) error {
	var envelope struct {
		Message  string            `json:"message"`
		Reason   string            `json:"reason"`
		Metadata map[string]string `json:"metadata"`
	}
	if err := json.Unmarshal(data, &envelope); err == nil && envelope.Message != "" {
		if envelope.Reason != "" {
			return fmt.Errorf("ccgo API error %d %s: %s", statusCode, envelope.Reason, envelope.Message)
		}
		return fmt.Errorf("ccgo API error %d: %s", statusCode, envelope.Message)
	}
	text := strings.TrimSpace(string(data))
	if text == "" {
		return fmt.Errorf("ccgo API error %d", statusCode)
	}
	return fmt.Errorf("ccgo API error %d: %s", statusCode, text)
}
