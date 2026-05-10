package projection

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/stretchr/testify/require"
)

func TestNodeFileOpsProxyToLocalBackend(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "README.md"), []byte("hello"), 0o644))
	backend, err := newLocalBackend(root)
	require.NoError(t, err)
	node := NewRootNode(backend)
	_ = fs.NewNodeFS(node, nil)
	ctx := context.Background()

	var attr fuse.AttrOut
	require.Equal(t, syscall.Errno(0), node.Getattr(ctx, nil, &attr))
	require.Equal(t, uint32(fuse.S_IFDIR|0o755), attr.Mode)

	stream, errno := node.Readdir(ctx)
	require.Equal(t, syscall.Errno(0), errno)
	names, errno := readAllDirStreamNames(stream)
	require.Equal(t, syscall.Errno(0), errno)
	require.Equal(t, []string{"README.md"}, names)

	child := &Node{backend: backend, relPath: "README.md"}
	buf := make([]byte, 16)
	readResult, errno := child.Read(ctx, nil, buf, 0)
	require.Equal(t, syscall.Errno(0), errno)
	data, status := readResult.Bytes(buf)
	require.Equal(t, fuse.OK, status)
	require.Equal(t, []byte("hello"), data)

	written, errno := child.Write(ctx, nil, []byte(" world"), 5)
	require.Equal(t, syscall.Errno(0), errno)
	require.Equal(t, uint32(6), written)
	content, err := os.ReadFile(filepath.Join(root, "README.md"))
	require.NoError(t, err)
	require.Equal(t, []byte("hello world"), content)
}

func TestNodeNamespaceOpsProxyToLocalBackend(t *testing.T) {
	root := t.TempDir()
	backend, err := newLocalBackend(root)
	require.NoError(t, err)
	node := NewRootNode(backend)
	_ = fs.NewNodeFS(node, nil)
	ctx := context.Background()

	var entry fuse.EntryOut
	_, errno := node.Mkdir(ctx, "dir", 0o755, &entry)
	require.Equal(t, syscall.Errno(0), errno)
	require.DirExists(t, filepath.Join(root, "dir"))

	_, _, _, errno = node.Create(ctx, "dir/file.txt", 0, 0o644, &entry)
	require.Equal(t, syscall.Errno(0), errno)
	require.FileExists(t, filepath.Join(root, "dir", "file.txt"))

	dirNode := &Node{backend: backend, relPath: "dir", isDir: true}
	errno = dirNode.Rename(ctx, "file.txt", dirNode, "renamed.txt", 0)
	require.Equal(t, syscall.Errno(0), errno)
	require.FileExists(t, filepath.Join(root, "dir", "renamed.txt"))

	fileNode := &Node{backend: backend, relPath: "dir/renamed.txt"}
	_, errno = fileNode.Write(ctx, nil, []byte("abcde"), 0)
	require.Equal(t, syscall.Errno(0), errno)
	var attr fuse.AttrOut
	errno = fileNode.Setattr(ctx, nil, &fuse.SetAttrIn{SetAttrInCommon: fuse.SetAttrInCommon{
		Valid: fuse.FATTR_SIZE,
		Size:  3,
	}}, &attr)
	require.Equal(t, syscall.Errno(0), errno)
	require.Equal(t, uint64(3), attr.Size)

	errno = dirNode.Unlink(ctx, "renamed.txt")
	require.Equal(t, syscall.Errno(0), errno)
	require.NoFileExists(t, filepath.Join(root, "dir", "renamed.txt"))
	errno = node.Rmdir(ctx, "dir")
	require.Equal(t, syscall.Errno(0), errno)
	require.NoDirExists(t, filepath.Join(root, "dir"))
}

func TestNodeReadlinkProxiesToLocalBackend(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.Symlink("README.md", filepath.Join(root, "link")))
	backend, err := newLocalBackend(root)
	require.NoError(t, err)
	linkNode := &Node{backend: backend, relPath: "link"}

	target, errno := linkNode.Readlink(context.Background())
	require.Equal(t, syscall.Errno(0), errno)
	require.Equal(t, []byte("README.md"), target)
}
