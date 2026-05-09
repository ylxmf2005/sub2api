package cli

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const defaultServerURL = "https://ccgo.example.com"
const deviceLoginStartPath = "/api/v1/ccgo/device-login/start"
const deviceLoginPollPath = "/api/v1/ccgo/device-login/poll"

type LoginOptions struct {
	Server       string
	Token        string
	User         string
	ConfigPath   string
	HTTPClient   *http.Client
	Output       io.Writer
	OpenBrowser  func(string) error
	PollTimeout  time.Duration
	PollInterval time.Duration
	DeviceID     string
	Now          func() time.Time
}

type LoginResult struct {
	ConfigPath    string
	Server        string
	User          string
	RedactedToken string
}

type deviceLoginStartResponse struct {
	DeviceCode      string    `json:"device_code"`
	UserCode        string    `json:"user_code"`
	VerificationURI string    `json:"verification_uri"`
	ExpiresAt       time.Time `json:"expires_at"`
	IntervalSeconds int       `json:"interval_seconds"`
}

type deviceLoginPollResponse struct {
	Status       string `json:"status"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
	TokenType    string `json:"token_type,omitempty"`
	UserID       *int64 `json:"user_id,omitempty"`
}

func Login(ctx context.Context, opts LoginOptions) (*LoginResult, error) {
	server := strings.TrimSpace(opts.Server)
	if server == "" {
		server = defaultServerURL
	}
	deviceID := strings.TrimSpace(opts.DeviceID)
	var err error
	if deviceID == "" {
		deviceID, err = randomDeviceID()
		if err != nil {
			return nil, err
		}
	}
	start, err := startDeviceLogin(ctx, opts.HTTPClient, server, deviceID)
	if err != nil {
		return nil, err
	}
	out := opts.Output
	if out != nil {
		fmt.Fprintf(out, "Open this URL to approve ccgo login:\n%s\n\nCode: %s\n", start.VerificationURI, start.UserCode)
	}
	if opts.OpenBrowser != nil {
		if err := opts.OpenBrowser(start.VerificationURI); err != nil {
			return nil, fmt.Errorf("open device login URL: %w", err)
		}
	} else if out == nil {
		_ = openBrowser(start.VerificationURI)
	}
	poll, err := pollDeviceLogin(ctx, opts.HTTPClient, server, start, opts)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(poll.AccessToken) == "" {
		return nil, fmt.Errorf("device login response is missing access token")
	}
	path := opts.ConfigPath
	if path == "" {
		path, err = DefaultConfigPath()
		if err != nil {
			return nil, err
		}
	}
	cfg := Config{
		Server:   server,
		User:     strings.TrimSpace(opts.User),
		Token:    poll.AccessToken,
		DeviceID: deviceID,
	}
	if err := SaveConfig(path, cfg); err != nil {
		return nil, err
	}
	return &LoginResult{
		ConfigPath:    path,
		Server:        server,
		User:          cfg.User,
		RedactedToken: RedactToken(poll.AccessToken),
	}, nil
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

func startDeviceLogin(ctx context.Context, client *http.Client, server, deviceID string) (*deviceLoginStartResponse, error) {
	payload := map[string]string{"device_id": deviceID}
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(payload); err != nil {
		return nil, fmt.Errorf("encode device login start request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, joinURL(server, deviceLoginStartPath), &body)
	if err != nil {
		return nil, fmt.Errorf("build device login start request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	data, err := doLoginJSON(client, req, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("start ccgo device login: %w", err)
	}
	var envelope struct {
		Code    int                      `json:"code"`
		Message string                   `json:"message"`
		Data    deviceLoginStartResponse `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("parse device login start response: %w", err)
	}
	if envelope.Code != 0 {
		return nil, fmt.Errorf("start ccgo device login: %s", envelope.Message)
	}
	if strings.TrimSpace(envelope.Data.DeviceCode) == "" || strings.TrimSpace(envelope.Data.VerificationURI) == "" {
		return nil, fmt.Errorf("device login start response is incomplete")
	}
	return &envelope.Data, nil
}

func pollDeviceLogin(ctx context.Context, client *http.Client, server string, start *deviceLoginStartResponse, opts LoginOptions) (*deviceLoginPollResponse, error) {
	if start == nil || strings.TrimSpace(start.DeviceCode) == "" {
		return nil, fmt.Errorf("device login response is incomplete")
	}
	timeout := opts.PollTimeout
	if timeout <= 0 {
		timeout = time.Until(start.ExpiresAt)
	}
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	pollInterval := opts.PollInterval
	if pollInterval <= 0 {
		pollInterval = time.Duration(start.IntervalSeconds) * time.Second
	}
	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}
	pollCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		result, pending, err := pollDeviceLoginOnce(pollCtx, client, server, start.DeviceCode)
		if err != nil {
			return nil, err
		}
		if !pending {
			return result, nil
		}
		timer := time.NewTimer(pollInterval)
		select {
		case <-pollCtx.Done():
			timer.Stop()
			return nil, fmt.Errorf("ccgo device login timed out: %w", pollCtx.Err())
		case <-timer.C:
		}
	}
}

func pollDeviceLoginOnce(ctx context.Context, client *http.Client, server, deviceCode string) (*deviceLoginPollResponse, bool, error) {
	payload := map[string]string{"device_code": deviceCode}
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(payload); err != nil {
		return nil, false, fmt.Errorf("encode device login poll request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, joinURL(server, deviceLoginPollPath), &body)
	if err != nil {
		return nil, false, fmt.Errorf("build device login poll request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	data, err := doLoginJSON(client, req, 30*time.Second)
	if err != nil {
		if strings.Contains(err.Error(), "CCGO_DEVICE_LOGIN_PENDING") {
			return nil, true, nil
		}
		return nil, false, fmt.Errorf("poll ccgo device login: %w", err)
	}
	var envelope struct {
		Code    int                     `json:"code"`
		Message string                  `json:"message"`
		Data    deviceLoginPollResponse `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, false, fmt.Errorf("parse device login poll response: %w", err)
	}
	if envelope.Code != 0 {
		if strings.Contains(envelope.Message, "pending") {
			return nil, true, nil
		}
		return nil, false, fmt.Errorf("poll ccgo device login: %s", envelope.Message)
	}
	return &envelope.Data, false, nil
}

func doLoginJSON(client *http.Client, req *http.Request, timeout time.Duration) ([]byte, error) {
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

func openBrowser(target string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", target)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		cmd = exec.Command("xdg-open", target)
	}
	return cmd.Start()
}
