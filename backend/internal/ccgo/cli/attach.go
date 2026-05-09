package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/protocol"
	coderws "github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func WorkstationAttachURL(server string, workspaceID int64) (string, error) {
	if workspaceID <= 0 {
		return "", fmt.Errorf("workspace id is required")
	}
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
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/api/v1/ccgo/workstations/" + strconv.FormatInt(workspaceID, 10) + "/attach"
	parsed.RawQuery = ""
	return parsed.String(), nil
}

type AttachOptions struct {
	ConfigPath  string
	Config      *Config
	WorkspaceID int64
	Input       io.Reader
	Output      io.Writer
	Dial        func(context.Context, string, *coderws.DialOptions) (*coderws.Conn, *http.Response, error)
}

func Attach(ctx context.Context, opts AttachOptions) error {
	if opts.WorkspaceID <= 0 {
		return fmt.Errorf("workspace id is required")
	}
	cfg := opts.Config
	var err error
	if cfg == nil {
		cfg, err = LoadConfig(opts.ConfigPath)
		if err != nil {
			return err
		}
	}
	target, err := WorkstationAttachURL(cfg.Server, opts.WorkspaceID)
	if err != nil {
		return err
	}
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+cfg.Token)
	dial := opts.Dial
	if dial == nil {
		dial = coderws.Dial
	}
	conn, resp, err := dial(ctx, target, &coderws.DialOptions{
		HTTPHeader:      headers,
		CompressionMode: coderws.CompressionDisabled,
	})
	if err != nil {
		return fmt.Errorf("attach ccgo workstation: %w%s", err, attachStatusSuffix(resp))
	}
	defer conn.CloseNow()
	conn.SetReadLimit(16 * 1024 * 1024)

	var writeMu sync.Mutex
	errCh := make(chan error, 2)
	if opts.Input != nil {
		go func() {
			buf := make([]byte, 32*1024)
			for {
				n, readErr := opts.Input.Read(buf)
				if n > 0 {
					writeMu.Lock()
					err := wsjson.Write(ctx, conn, protocol.Envelope{Type: protocol.MessageTypeTerminalInput, Payload: append([]byte(nil), buf[:n]...)})
					writeMu.Unlock()
					if err != nil {
						errCh <- err
						return
					}
				}
				if readErr != nil {
					if readErr == io.EOF {
						errCh <- nil
						return
					}
					errCh <- readErr
					return
				}
			}
		}()
	}
	go func() {
		for {
			var env protocol.Envelope
			if err := wsjson.Read(ctx, conn, &env); err != nil {
				errCh <- err
				return
			}
			if env.Type != protocol.MessageTypeTerminalOutput || opts.Output == nil {
				continue
			}
			if _, err := opts.Output.Write(env.Payload); err != nil {
				errCh <- err
				return
			}
		}
	}()
	err = <-errCh
	if err == nil {
		return nil
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}

func attachStatusSuffix(resp *http.Response) string {
	if resp == nil {
		return ""
	}
	return fmt.Sprintf(" (status %d)", resp.StatusCode)
}
