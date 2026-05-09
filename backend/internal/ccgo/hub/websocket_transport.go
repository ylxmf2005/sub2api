package hub

import (
	"context"
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/protocol"
	coderws "github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type WebSocketTransport struct {
	conn    *coderws.Conn
	writeMu sync.Mutex
}

func NewWebSocketTransport(conn *coderws.Conn) *WebSocketTransport {
	return &WebSocketTransport{conn: conn}
}

func (t *WebSocketTransport) Send(ctx context.Context, env protocol.Envelope) error {
	t.writeMu.Lock()
	defer t.writeMu.Unlock()
	return wsjson.Write(ctx, t.conn, env)
}

func (t *WebSocketTransport) Recv(ctx context.Context) (protocol.Envelope, error) {
	var env protocol.Envelope
	err := wsjson.Read(ctx, t.conn, &env)
	return env, err
}

func (t *WebSocketTransport) Close() error {
	if t == nil || t.conn == nil {
		return nil
	}
	return t.conn.CloseNow()
}
