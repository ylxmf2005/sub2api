package agent

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/protocol"
	coderws "github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

const agentConnectPath = "/api/v1/ccgo/agent/connect"

type ConnectorOptions struct {
	Server string
	Token  string
	Nonce  string
	Root   string

	Dial func(context.Context, string, *coderws.DialOptions) (*coderws.Conn, *http.Response, error)
}

type Connector struct {
	agent *Agent
}

func NewConnector(agent *Agent) (*Connector, error) {
	if agent == nil {
		return nil, fmt.Errorf("ccgo agent is required")
	}
	return &Connector{agent: agent}, nil
}

func RunConnector(ctx context.Context, opts ConnectorOptions) error {
	localAgent, err := New(opts.Root)
	if err != nil {
		return err
	}
	connector, err := NewConnector(localAgent)
	if err != nil {
		return err
	}
	return connector.Run(ctx, opts)
}

func (c *Connector) Run(ctx context.Context, opts ConnectorOptions) error {
	if c == nil || c.agent == nil {
		return fmt.Errorf("ccgo agent is required")
	}
	token := strings.TrimSpace(opts.Token)
	nonce := strings.TrimSpace(opts.Nonce)
	if token == "" || nonce == "" {
		return fmt.Errorf("ccgo agent credential token and nonce are required")
	}
	target, err := AgentConnectURL(opts.Server)
	if err != nil {
		return err
	}
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+token)
	headers.Set("X-CCGO-Nonce", nonce)
	dial := opts.Dial
	if dial == nil {
		dial = coderws.Dial
	}
	wsConn, resp, err := dial(ctx, target, &coderws.DialOptions{
		HTTPHeader:      headers,
		CompressionMode: coderws.CompressionDisabled,
	})
	if err != nil {
		return fmt.Errorf("connect ccgo agent websocket: %w%s", err, responseStatusSuffix(resp))
	}
	defer wsConn.CloseNow()
	wsConn.SetReadLimit(16 * 1024 * 1024)
	var writeMu sync.Mutex
	heartbeatDone := make(chan struct{})
	go sendHeartbeats(ctx, wsConn, &writeMu, heartbeatDone)
	defer close(heartbeatDone)
	for {
		var request protocol.Envelope
		if err := wsjson.Read(ctx, wsConn, &request); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("read ccgo agent request: %w", err)
		}
		response, err := c.agent.Handle(ctx, request)
		if err != nil {
			response = protocol.NewErrorResponse(request.RequestID, protocolError(err))
		}
		writeMu.Lock()
		err = wsjson.Write(ctx, wsConn, response)
		writeMu.Unlock()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("write ccgo agent response: %w", err)
		}
	}
}

func AgentConnectURL(server string) (string, error) {
	server = strings.TrimSpace(server)
	if server == "" {
		return "", fmt.Errorf("ccgo server is required")
	}
	parsed, err := url.Parse(server)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid ccgo server URL")
	}
	switch parsed.Scheme {
	case "http":
		parsed.Scheme = "ws"
	case "https":
		parsed.Scheme = "wss"
	case "ws", "wss":
	default:
		return "", fmt.Errorf("unsupported ccgo server URL scheme %q", parsed.Scheme)
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + agentConnectPath
	parsed.RawQuery = ""
	return parsed.String(), nil
}

func sendHeartbeats(ctx context.Context, conn *coderws.Conn, writeMu *sync.Mutex, done <-chan struct{}) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-done:
			return
		case <-ticker.C:
			writeMu.Lock()
			_ = wsjson.Write(ctx, conn, protocol.Envelope{Type: protocol.MessageTypeHeartbeat})
			writeMu.Unlock()
		}
	}
}

func responseStatusSuffix(resp *http.Response) string {
	if resp == nil {
		return ""
	}
	return fmt.Sprintf(" (status %d)", resp.StatusCode)
}
