package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/protocol"
	"github.com/stretchr/testify/require"
)

func TestFileServiceReadWriteListStatInsideRoot(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, "dir"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "dir", "existing.txt"), []byte("old"), 0o644))
	localRoot, err := NewLocalRoot(root)
	require.NoError(t, err)
	files := NewFileService(localRoot)
	ctx := context.Background()

	writeResp, err := files.Write(ctx, protocol.FileWriteRequest{Path: "dir/new.txt", Data: []byte("hello"), Truncate: true})
	require.NoError(t, err)
	require.Equal(t, 5, writeResp.BytesWritten)

	readResp, err := files.Read(ctx, protocol.FileReadRequest{Path: "dir/new.txt"})
	require.NoError(t, err)
	require.Equal(t, []byte("hello"), readResp.Data)
	require.True(t, readResp.EOF)

	statResp, err := files.Stat(ctx, protocol.FileStatRequest{Path: "dir/new.txt"})
	require.NoError(t, err)
	require.Equal(t, "dir/new.txt", statResp.Path)
	require.False(t, statResp.IsDir)
	require.Equal(t, int64(5), statResp.Size)

	listResp, err := files.List(ctx, protocol.FileListRequest{Path: "dir"})
	require.NoError(t, err)
	require.Equal(t, []string{"existing.txt", "new.txt"}, fileListNames(listResp.Entries))
}

func TestFileServiceRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	localRoot, err := NewLocalRoot(root)
	require.NoError(t, err)
	files := NewFileService(localRoot)

	_, err = files.Read(context.Background(), protocol.FileReadRequest{Path: "../secret.txt"})
	require.Error(t, err)
	require.Contains(t, err.Error(), protocol.ErrorPathOutsideRoot)
}

func TestFileServiceRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o644))
	require.NoError(t, os.Symlink(outside, filepath.Join(root, "link")))
	localRoot, err := NewLocalRoot(root)
	require.NoError(t, err)
	files := NewFileService(localRoot)

	_, err = files.Read(context.Background(), protocol.FileReadRequest{Path: "link/secret.txt"})
	require.Error(t, err)
	require.Contains(t, err.Error(), protocol.ErrorPathOutsideRoot)
}

func TestAgentHandleReturnsProtocolErrorEnvelope(t *testing.T) {
	root := t.TempDir()
	a, err := New(root)
	require.NoError(t, err)
	req, err := protocol.NewRequest("req-1", protocol.MethodFileRead, protocol.FileReadRequest{Path: "../secret.txt"})
	require.NoError(t, err)

	resp, err := a.Handle(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, protocol.MessageTypeResponse, resp.Type)
	require.Equal(t, "req-1", resp.RequestID)
	require.NotNil(t, resp.Error)
	require.Equal(t, protocol.ErrorPathOutsideRoot, resp.Error.Code)
}

func fileListNames(entries []protocol.FileListEntry) []string {
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name)
	}
	return names
}
