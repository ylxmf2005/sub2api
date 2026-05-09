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
	data, err := c.doJSON(req, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("resolve ccgo workspace: %w", err)
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
	ServerPID   string `json:"server_pid,omitempty"`
	StopReason  string `json:"stop_reason,omitempty"`
}

type StartWorkstationResult struct {
	Workspace Workspace      `json:"workspace"`
	Run       WorkstationRun `json:"run"`
	Reused    bool           `json:"reused"`
}

type RunnerStatus struct {
	WorkspaceID       int64     `json:"workspace_id"`
	RunID             string    `json:"run_id,omitempty"`
	ServerPID         string    `json:"server_pid,omitempty"`
	Running           bool      `json:"running"`
	ProjectionMounted bool      `json:"projection_mounted"`
	ExecBridgeRunning bool      `json:"exec_bridge_running"`
	TerminalReady     bool      `json:"terminal_ready"`
	LastCheckedAt     time.Time `json:"last_checked_at"`
}

type WorkstationStatusResult struct {
	Workspace      Workspace       `json:"workspace"`
	AgentConnected bool            `json:"agent_connected"`
	AgentError     string          `json:"agent_error,omitempty"`
	Runner         RunnerStatus    `json:"runner"`
	LatestRun      *WorkstationRun `json:"latest_run,omitempty"`
	Resumable      bool            `json:"resumable"`
}

type StopWorkstationResult struct {
	Workspace         Workspace       `json:"workspace"`
	Run               *WorkstationRun `json:"run,omitempty"`
	Runner            RunnerStatus    `json:"runner"`
	Stopped           bool            `json:"stopped"`
	AgentDisconnected bool            `json:"agent_disconnected"`
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
	data, err := c.doJSON(req, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("start ccgo workstation: %w", err)
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

func (c WorkspaceClient) WorkstationStatus(ctx context.Context, workspaceID int64) (*WorkstationStatusResult, error) {
	if workspaceID <= 0 {
		return nil, fmt.Errorf("workspace id is required")
	}
	cfg, err := c.loadConfig()
	if err != nil {
		return nil, fmt.Errorf("status ccgo workstation: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, joinURL(cfg.Server, workstationPath(workspaceID, "status")), nil)
	if err != nil {
		return nil, fmt.Errorf("build workstation status request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	data, err := c.doJSON(req, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("status ccgo workstation: %w", err)
	}
	var envelope struct {
		Code    int                     `json:"code"`
		Message string                  `json:"message"`
		Data    WorkstationStatusResult `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("parse workstation status response: %w", err)
	}
	if envelope.Code != 0 {
		return nil, fmt.Errorf("workstation status: %s", envelope.Message)
	}
	if envelope.Data.Workspace.ID <= 0 {
		return nil, fmt.Errorf("workstation status response is incomplete")
	}
	return &envelope.Data, nil
}

func (c WorkspaceClient) StopWorkstation(ctx context.Context, workspaceID int64, reason string) (*StopWorkstationResult, error) {
	if workspaceID <= 0 {
		return nil, fmt.Errorf("workspace id is required")
	}
	cfg, err := c.loadConfig()
	if err != nil {
		return nil, fmt.Errorf("stop ccgo workstation: %w", err)
	}
	payload := map[string]string{"reason": strings.TrimSpace(reason)}
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(payload); err != nil {
		return nil, fmt.Errorf("encode workstation stop request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, joinURL(cfg.Server, workstationPath(workspaceID, "stop")), &body)
	if err != nil {
		return nil, fmt.Errorf("build workstation stop request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	data, err := c.doJSON(req, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("stop ccgo workstation: %w", err)
	}
	var envelope struct {
		Code    int                   `json:"code"`
		Message string                `json:"message"`
		Data    StopWorkstationResult `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("parse workstation stop response: %w", err)
	}
	if envelope.Code != 0 {
		return nil, fmt.Errorf("stop ccgo workstation: %s", envelope.Message)
	}
	if envelope.Data.Workspace.ID <= 0 {
		return nil, fmt.Errorf("workstation stop response is incomplete")
	}
	return &envelope.Data, nil
}

func (c WorkspaceClient) loadConfig() (*Config, error) {
	if c.Config != nil {
		return c.Config, nil
	}
	return LoadConfig(c.ConfigPath)
}

func (c WorkspaceClient) doJSON(req *http.Request, timeout time.Duration) ([]byte, error) {
	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, decodeAPIError(resp.StatusCode, data)
	}
	return data, nil
}

func workstationPath(workspaceID int64, action string) string {
	return "/api/v1/ccgo/workstations/" + fmt.Sprintf("%d", workspaceID) + "/" + strings.TrimLeft(action, "/")
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
