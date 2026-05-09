package hub

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/protocol"
	"github.com/stretchr/testify/require"
)

type memoryTransport struct {
	sent   chan protocol.Envelope
	recv   chan protocol.Envelope
	closed chan struct{}
	once   sync.Once
}

func newMemoryTransport() *memoryTransport {
	return &memoryTransport{
		sent:   make(chan protocol.Envelope, 8),
		recv:   make(chan protocol.Envelope, 8),
		closed: make(chan struct{}),
	}
}

func (t *memoryTransport) Send(_ context.Context, env protocol.Envelope) error {
	select {
	case <-t.closed:
		return protocol.NewError(protocol.ErrorAgentDisconnected, "closed")
	case t.sent <- env:
		return nil
	}
}

func (t *memoryTransport) Recv(ctx context.Context) (protocol.Envelope, error) {
	select {
	case <-ctx.Done():
		return protocol.Envelope{}, ctx.Err()
	case <-t.closed:
		return protocol.Envelope{}, protocol.NewError(protocol.ErrorAgentDisconnected, "closed")
	case env := <-t.recv:
		return env, nil
	}
}

func (t *memoryTransport) Close() error {
	t.once.Do(func() { close(t.closed) })
	return nil
}

func TestConnectionManager_RequestRoutesResponseByID(t *testing.T) {
	manager := NewConnectionManager()
	transport := newMemoryTransport()
	conn := manager.Register(123, transport)

	done := make(chan error, 1)
	var out protocol.FileStatResponse
	go func() {
		done <- conn.Request(context.Background(), protocol.MethodFileStat, protocol.FileStatRequest{Path: "file.txt"}, &out)
	}()

	sent := <-transport.sent
	require.Equal(t, protocol.MethodFileStat, sent.Method)
	require.NotEmpty(t, sent.RequestID)
	resp, err := protocol.NewResponse(sent.RequestID, protocol.FileStatResponse{Path: "file.txt", Size: 12})
	require.NoError(t, err)
	transport.recv <- resp

	require.NoError(t, <-done)
	require.Equal(t, "file.txt", out.Path)
	require.Equal(t, int64(12), out.Size)
}

func TestConnectionManager_RequestFailsWhenDisconnected(t *testing.T) {
	manager := NewConnectionManager()
	transport := newMemoryTransport()
	conn := manager.Register(123, transport)

	done := make(chan error, 1)
	go func() {
		var out protocol.FileStatResponse
		done <- conn.Request(context.Background(), protocol.MethodFileStat, protocol.FileStatRequest{Path: "file.txt"}, &out)
	}()
	<-transport.sent
	manager.Unregister(123)

	err := <-done
	require.Error(t, err)
	require.Contains(t, err.Error(), protocol.ErrorAgentDisconnected)
}

func TestConnectionManager_RequestHonorsContextTimeout(t *testing.T) {
	manager := NewConnectionManager()
	transport := newMemoryTransport()
	conn := manager.Register(123, transport)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	var out protocol.FileStatResponse
	err := conn.Request(ctx, protocol.MethodFileStat, protocol.FileStatRequest{Path: "file.txt"}, &out)
	require.Error(t, err)
	require.Contains(t, err.Error(), protocol.ErrorRequestTimeout)
}

func TestConnectionManager_RPCFailsWhenAgentMissing(t *testing.T) {
	manager := NewConnectionManager()

	_, err := manager.FileStat(context.Background(), 123, protocol.FileStatRequest{Path: "file.txt"})
	require.Error(t, err)
	require.Contains(t, err.Error(), protocol.ErrorAgentDisconnected)
}
