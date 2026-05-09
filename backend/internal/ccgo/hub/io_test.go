package hub

import (
	"bytes"
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/protocol"
	"github.com/stretchr/testify/require"
)

func TestEnvelopeReaderReadsTerminalInputPayloads(t *testing.T) {
	frames := []protocol.Envelope{
		{Type: protocol.MessageTypeHeartbeat},
		{Type: protocol.MessageTypeTerminalInput, Payload: []byte("abc")},
	}
	reader := NewEnvelopeReader(context.Background(), func(context.Context) (protocol.Envelope, error) {
		frame := frames[0]
		frames = frames[1:]
		return frame, nil
	})

	buf := make([]byte, 2)
	n, err := reader.Read(buf)
	require.NoError(t, err)
	require.Equal(t, 2, n)
	require.Equal(t, "ab", string(buf))

	n, err = reader.Read(buf)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	require.Equal(t, "c", string(buf[:n]))
}

func TestEnvelopeWriterWritesTerminalOutputFrames(t *testing.T) {
	var sent protocol.Envelope
	writer := NewEnvelopeWriter(context.Background(), func(ctx context.Context, env protocol.Envelope) error {
		sent = env
		return nil
	})

	n, err := writer.Write([]byte("hello"))
	require.NoError(t, err)
	require.Equal(t, 5, n)
	require.Equal(t, protocol.MessageTypeTerminalOutput, sent.Type)
	require.Equal(t, "hello", string(sent.Payload))

	var out bytes.Buffer
	_, _ = out.Write(sent.Payload)
	require.Equal(t, "hello", out.String())
}
