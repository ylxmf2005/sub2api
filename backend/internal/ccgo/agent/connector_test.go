package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/protocol"
	coderws "github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/require"
)

func TestAgentConnectURL(t *testing.T) {
	got, err := AgentConnectURL("https://ccgo.example.com/base/")
	require.NoError(t, err)
	require.Equal(t, "wss://ccgo.example.com/base/api/v1/ccgo/agent/connect", got)

	got, err = AgentConnectURL("http://127.0.0.1:3000")
	require.NoError(t, err)
	require.Equal(t, "ws://127.0.0.1:3000/api/v1/ccgo/agent/connect", got)

	_, err = AgentConnectURL("ftp://ccgo.example.com")
	require.ErrorContains(t, err, "unsupported")
}

func TestConnectorHandlesFileRequestOverWebSocket(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "hello.txt"), []byte("hello"), 0o644))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	requestHeaders := make(chan http.Header, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestHeaders <- r.Header.Clone()
		conn, err := coderws.Accept(w, r, &coderws.AcceptOptions{CompressionMode: coderws.CompressionDisabled})
		require.NoError(t, err)
		defer conn.CloseNow()
		req, err := protocol.NewRequest("req-1", protocol.MethodFileRead, protocol.FileReadRequest{
			Path:  "hello.txt",
			Limit: 32,
		})
		require.NoError(t, err)
		require.NoError(t, wsjson.Write(ctx, conn, req))
		var resp protocol.Envelope
		require.NoError(t, wsjson.Read(ctx, conn, &resp))
		require.Equal(t, protocol.MessageTypeResponse, resp.Type)
		require.Equal(t, "req-1", resp.RequestID)
		data, err := protocol.DecodePayload[protocol.FileReadResponse](resp)
		require.NoError(t, err)
		require.Equal(t, []byte("hello"), data.Data)
		_ = conn.Close(coderws.StatusNormalClosure, "done")
	}))
	defer server.Close()

	errCh := make(chan error, 1)
	go func() {
		errCh <- RunConnector(ctx, ConnectorOptions{
			Server: server.URL,
			Token:  "agent_token",
			Nonce:  "agent_nonce",
			Root:   root,
		})
	}()

	headers := <-requestHeaders
	require.Equal(t, "Bearer agent_token", headers.Get("Authorization"))
	require.Equal(t, "agent_nonce", headers.Get("X-CCGO-Nonce"))
	err := <-errCh
	require.Error(t, err)
	require.Contains(t, err.Error(), "read ccgo agent request")
}

func TestConnectorRejectsMissingCredentialBeforeDial(t *testing.T) {
	err := RunConnector(context.Background(), ConnectorOptions{
		Server: "https://ccgo.example.com",
		Root:   t.TempDir(),
	})
	require.ErrorContains(t, err, "credential")
}

func TestConnectorFileRequestStaysRootConfined(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret.txt")
	require.NoError(t, os.WriteFile(outside, []byte("secret"), 0o644))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := coderws.Accept(w, r, &coderws.AcceptOptions{CompressionMode: coderws.CompressionDisabled})
		require.NoError(t, err)
		defer conn.CloseNow()
		req, err := protocol.NewRequest("req-escape", protocol.MethodFileRead, protocol.FileReadRequest{
			Path: filepath.Join("..", filepath.Base(filepath.Dir(outside)), filepath.Base(outside)),
		})
		require.NoError(t, err)
		require.NoError(t, wsjson.Write(ctx, conn, req))
		var resp protocol.Envelope
		require.NoError(t, wsjson.Read(ctx, conn, &resp))
		require.NotNil(t, resp.Error)
		require.Equal(t, protocol.ErrorPathOutsideRoot, resp.Error.Code)
		_ = conn.Close(coderws.StatusNormalClosure, "done")
	}))
	defer server.Close()

	errCh := make(chan error, 1)
	go func() {
		errCh <- RunConnector(ctx, ConnectorOptions{
			Server: server.URL,
			Token:  "agent_token",
			Nonce:  "agent_nonce",
			Root:   root,
		})
	}()
	err := <-errCh
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "read ccgo agent request") || strings.Contains(err.Error(), "closed"))
}
