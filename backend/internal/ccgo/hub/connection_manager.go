package hub

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/protocol"
	"github.com/google/uuid"
)

type Transport interface {
	Send(context.Context, protocol.Envelope) error
	Recv(context.Context) (protocol.Envelope, error)
	Close() error
}

type ConnectionManager struct {
	mu          sync.RWMutex
	connections map[int64]*Connection
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{connections: make(map[int64]*Connection)}
}

func ProvideConnectionManager() *ConnectionManager {
	return NewConnectionManager()
}

func (m *ConnectionManager) Register(workspaceID int64, transport Transport) *Connection {
	conn := newConnection(workspaceID, transport)
	m.mu.Lock()
	old := m.connections[workspaceID]
	m.connections[workspaceID] = conn
	m.mu.Unlock()
	if old != nil {
		old.closeWithError(protocol.NewError(protocol.ErrorAgentDisconnected, "agent connection replaced"))
	}
	go conn.readLoop()
	return conn
}

func (m *ConnectionManager) Get(workspaceID int64) (*Connection, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	conn, ok := m.connections[workspaceID]
	if !ok || conn.isClosed() {
		return nil, false
	}
	return conn, true
}

func (m *ConnectionManager) Unregister(workspaceID int64) {
	m.mu.Lock()
	conn := m.connections[workspaceID]
	delete(m.connections, workspaceID)
	m.mu.Unlock()
	if conn != nil {
		conn.closeWithError(protocol.NewError(protocol.ErrorAgentDisconnected, "agent disconnected"))
	}
}

type Connection struct {
	workspaceID int64
	transport   Transport
	pendingMu   sync.Mutex
	pending     map[string]chan protocol.Envelope
	closed      chan struct{}
	closeOnce   sync.Once
	closeErr    *protocol.Error
	lastSeenMu  sync.RWMutex
	lastSeen    time.Time
}

func newConnection(workspaceID int64, transport Transport) *Connection {
	return &Connection{
		workspaceID: workspaceID,
		transport:   transport,
		pending:     make(map[string]chan protocol.Envelope),
		closed:      make(chan struct{}),
		lastSeen:    time.Now(),
	}
}

func (c *Connection) LastSeen() time.Time {
	c.lastSeenMu.RLock()
	defer c.lastSeenMu.RUnlock()
	return c.lastSeen
}

func (c *Connection) Request(ctx context.Context, method string, payload any, out any) error {
	if c.isClosed() {
		return c.closeErr
	}
	requestID := uuid.NewString()
	env, err := protocol.NewRequest(requestID, method, payload)
	if err != nil {
		return err
	}
	ch := make(chan protocol.Envelope, 1)
	c.pendingMu.Lock()
	c.pending[requestID] = ch
	c.pendingMu.Unlock()
	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, requestID)
		c.pendingMu.Unlock()
	}()
	if err := c.transport.Send(ctx, env); err != nil {
		c.closeWithError(protocol.NewError(protocol.ErrorAgentDisconnected, err.Error()))
		return err
	}
	select {
	case <-ctx.Done():
		return protocol.NewError(protocol.ErrorRequestTimeout, ctx.Err().Error())
	case <-c.closed:
		return c.closeErr
	case response := <-ch:
		if response.Error != nil {
			return response.Error
		}
		if out == nil || len(response.Payload) == 0 {
			return nil
		}
		return json.Unmarshal(response.Payload, out)
	}
}

func (c *Connection) readLoop() {
	for {
		env, err := c.transport.Recv(context.Background())
		if err != nil {
			c.closeWithError(protocol.NewError(protocol.ErrorAgentDisconnected, err.Error()))
			return
		}
		c.lastSeenMu.Lock()
		c.lastSeen = time.Now()
		c.lastSeenMu.Unlock()
		if env.Type == protocol.MessageTypeHeartbeat {
			continue
		}
		if env.Type != protocol.MessageTypeResponse || env.RequestID == "" {
			continue
		}
		c.pendingMu.Lock()
		ch := c.pending[env.RequestID]
		c.pendingMu.Unlock()
		if ch != nil {
			ch <- env
		}
	}
}

func (c *Connection) isClosed() bool {
	select {
	case <-c.closed:
		return true
	default:
		return false
	}
}

func (c *Connection) closeWithError(err *protocol.Error) {
	c.closeOnce.Do(func() {
		c.closeErr = err
		close(c.closed)
		_ = c.transport.Close()
		c.pendingMu.Lock()
		for id, ch := range c.pending {
			delete(c.pending, id)
			ch <- protocol.NewErrorResponse(id, err)
		}
		c.pendingMu.Unlock()
	})
}
