package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/protocol"
	"github.com/Wei-Shaw/sub2api/internal/ccgo/shellwrapper"
)

type ExecRequester interface {
	Exec(context.Context, int64, protocol.ExecRequest) (protocol.ExecResponse, error)
}

type ExecBridge struct {
	socketPath string
	listener   net.Listener
	requester  ExecRequester
	closeOnce  sync.Once
	closed     chan struct{}
}

func StartExecBridge(socketPath string, requester ExecRequester) (*ExecBridge, error) {
	if socketPath == "" {
		return nil, fmt.Errorf("ccgo exec bridge socket path is required")
	}
	if requester == nil {
		return nil, fmt.Errorf("ccgo exec requester is required")
	}
	if err := os.MkdirAll(filepath.Dir(socketPath), 0o700); err != nil {
		return nil, fmt.Errorf("create ccgo exec bridge dir: %w", err)
	}
	if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("remove stale ccgo exec bridge socket: %w", err)
	}
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("listen ccgo exec bridge: %w", err)
	}
	bridge := &ExecBridge{
		socketPath: socketPath,
		listener:   listener,
		requester:  requester,
		closed:     make(chan struct{}),
	}
	go bridge.serve()
	return bridge, nil
}

func (b *ExecBridge) SocketPath() string {
	if b == nil {
		return ""
	}
	return b.socketPath
}

func (b *ExecBridge) Close() error {
	if b == nil {
		return nil
	}
	var err error
	b.closeOnce.Do(func() {
		close(b.closed)
		err = b.listener.Close()
		_ = os.Remove(b.socketPath)
	})
	return err
}

func (b *ExecBridge) serve() {
	for {
		conn, err := b.listener.Accept()
		if err != nil {
			select {
			case <-b.closed:
				return
			default:
				return
			}
		}
		go b.handle(conn)
	}
}

func (b *ExecBridge) handle(conn net.Conn) {
	defer conn.Close()
	var req shellwrapper.ExecBridgeRequest
	if err := json.NewDecoder(conn).Decode(&req); err != nil {
		_ = json.NewEncoder(conn).Encode(shellwrapper.ExecBridgeResponse{
			ExitCode: 2,
			Error:    protocol.NewError(protocol.ErrorInvalidRequest, err.Error()),
		})
		return
	}
	if req.WorkspaceID <= 0 || req.Command == "" || req.Cwd == "" {
		_ = json.NewEncoder(conn).Encode(shellwrapper.ExecBridgeResponse{
			ExitCode: 2,
			Error:    protocol.NewError(protocol.ErrorInvalidRequest, "workspace id, cwd, and command are required"),
		})
		return
	}
	resp, err := b.requester.Exec(context.Background(), req.WorkspaceID, protocol.ExecRequest{
		Cwd:       req.Cwd,
		Command:   req.Command,
		TimeoutMS: req.TimeoutMS,
	})
	out := shellwrapper.ExecBridgeResponse{
		Stdout:   resp.Stdout,
		Stderr:   resp.Stderr,
		ExitCode: resp.ExitCode,
	}
	if err != nil {
		out.Error = protocolError(err)
		if out.ExitCode == 0 {
			out.ExitCode = 1
		}
	}
	_ = json.NewEncoder(conn).Encode(out)
}

func protocolError(err error) *protocol.Error {
	if err == nil {
		return nil
	}
	if ccgoErr, ok := err.(*protocol.Error); ok {
		return ccgoErr
	}
	return protocol.NewError(protocol.ErrorInternal, err.Error())
}
