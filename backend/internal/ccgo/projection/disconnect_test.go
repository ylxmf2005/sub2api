package projection

import (
	"context"
	"syscall"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/protocol"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/stretchr/testify/require"
)

type disconnectedBackend struct{}

func (disconnectedBackend) err() error {
	return protocol.NewError(protocol.ErrorAgentDisconnected, "agent is not connected")
}

func (b disconnectedBackend) Stat(context.Context, string) (protocol.FileStatResponse, error) {
	return protocol.FileStatResponse{}, b.err()
}
func (b disconnectedBackend) Read(context.Context, string, int64, int64) (protocol.FileReadResponse, error) {
	return protocol.FileReadResponse{}, b.err()
}
func (b disconnectedBackend) Write(context.Context, string, []byte, int64, bool) (protocol.FileWriteResponse, error) {
	return protocol.FileWriteResponse{}, b.err()
}
func (b disconnectedBackend) List(context.Context, string) (protocol.FileListResponse, error) {
	return protocol.FileListResponse{}, b.err()
}
func (b disconnectedBackend) Mkdir(context.Context, string, uint32) (protocol.FileStatResponse, error) {
	return protocol.FileStatResponse{}, b.err()
}
func (b disconnectedBackend) Remove(context.Context, string, bool) error {
	return b.err()
}
func (b disconnectedBackend) Rename(context.Context, string, string) error {
	return b.err()
}
func (b disconnectedBackend) Truncate(context.Context, string, int64) (protocol.FileStatResponse, error) {
	return protocol.FileStatResponse{}, b.err()
}
func (b disconnectedBackend) Chmod(context.Context, string, uint32) (protocol.FileStatResponse, error) {
	return protocol.FileStatResponse{}, b.err()
}

func TestNodeDisconnectReturnsIOError(t *testing.T) {
	node := NewRootNode(disconnectedBackend{})
	var attr fuse.AttrOut
	errno := node.Getattr(context.Background(), nil, &attr)
	require.Equal(t, syscall.EIO, errno)

	_, errno = node.Readdir(context.Background())
	require.Equal(t, syscall.EIO, errno)
}
