package hub

import (
	"context"
	"io"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/protocol"
)

type EnvelopeReader struct {
	ctx       context.Context
	recv      func(context.Context) (protocol.Envelope, error)
	buffer    []byte
	closeCode string
}

func NewEnvelopeReader(ctx context.Context, recv func(context.Context) (protocol.Envelope, error)) *EnvelopeReader {
	return &EnvelopeReader{ctx: ctx, recv: recv}
}

func (r *EnvelopeReader) Read(p []byte) (int, error) {
	for len(r.buffer) == 0 {
		env, err := r.recv(r.ctx)
		if err != nil {
			return 0, err
		}
		if env.Type == protocol.MessageTypeHeartbeat {
			continue
		}
		if env.Type != protocol.MessageTypeTerminalInput {
			continue
		}
		r.buffer = append(r.buffer[:0], env.Payload...)
	}
	n := copy(p, r.buffer)
	r.buffer = r.buffer[n:]
	return n, nil
}

type EnvelopeWriter struct {
	ctx  context.Context
	send func(context.Context, protocol.Envelope) error
}

func NewEnvelopeWriter(ctx context.Context, send func(context.Context, protocol.Envelope) error) *EnvelopeWriter {
	return &EnvelopeWriter{ctx: ctx, send: send}
}

func (w *EnvelopeWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if err := w.send(w.ctx, protocol.Envelope{Type: protocol.MessageTypeTerminalOutput, Payload: append([]byte(nil), p...)}); err != nil {
		return 0, err
	}
	return len(p), nil
}

var _ io.Reader = (*EnvelopeReader)(nil)
var _ io.Writer = (*EnvelopeWriter)(nil)
