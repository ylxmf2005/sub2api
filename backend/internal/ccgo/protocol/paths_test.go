package protocol

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveInsideRoot_AllowsRelativePath(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, "dir"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "dir", "file.txt"), []byte("x"), 0o644))

	resolved, err := ResolveInsideRoot(root, "dir/file.txt")
	require.NoError(t, err)
	expected, err := filepath.EvalSymlinks(filepath.Join(root, "dir", "file.txt"))
	require.NoError(t, err)
	require.Equal(t, expected, resolved)
}

func TestResolveInsideRoot_RejectsTraversal(t *testing.T) {
	root := t.TempDir()
	_, err := ResolveInsideRoot(root, "../outside")
	require.Error(t, err)
	require.Contains(t, err.Error(), ErrorPathOutsideRoot)
}

func TestResolveInsideRoot_RejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("x"), 0o644))
	link := filepath.Join(root, "link")
	require.NoError(t, os.Symlink(outside, link))

	_, err := ResolveInsideRoot(root, "link/secret.txt")
	require.Error(t, err)
	require.Contains(t, err.Error(), ErrorPathOutsideRoot)
}

func TestResolveInsideRootNoFollow_AllowsSymlinkNodeInsideRoot(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(root, "link")
	require.NoError(t, os.Symlink(outside, link))

	resolved, err := ResolveInsideRootNoFollow(root, "link")
	require.NoError(t, err)
	expectedRoot, err := filepath.EvalSymlinks(root)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(expectedRoot, "link"), resolved)
}

func TestResolveInsideRootNoFollow_RejectsParentSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(root, "link")
	require.NoError(t, os.Symlink(outside, link))

	_, err := ResolveInsideRootNoFollow(root, "link/child")
	require.Error(t, err)
	require.Contains(t, err.Error(), ErrorPathOutsideRoot)
}

func TestResolveInsideRoot_AllowsMissingLeafInsideRoot(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, "dir"), 0o755))

	resolved, err := ResolveInsideRoot(root, "dir/new.txt")
	require.NoError(t, err)
	expectedDir, err := filepath.EvalSymlinks(filepath.Join(root, "dir"))
	require.NoError(t, err)
	require.Equal(t, filepath.Join(expectedDir, "new.txt"), resolved)
}

func TestResolveInsideRoot_RejectsMissingLeafThroughSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(root, "link")
	require.NoError(t, os.Symlink(outside, link))

	_, err := ResolveInsideRoot(root, "link/new.txt")
	require.Error(t, err)
	require.Contains(t, err.Error(), ErrorPathOutsideRoot)
}

func TestResolveLocalPathInsideRoot_AllowsAbsolutePathInsideRoot(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "file.txt"), []byte("x"), 0o644))

	resolved, err := ResolveLocalPathInsideRoot(root, filepath.Join(root, "file.txt"))
	require.NoError(t, err)
	expected, err := filepath.EvalSymlinks(filepath.Join(root, "file.txt"))
	require.NoError(t, err)
	require.Equal(t, expected, resolved)
}

func TestResolveLocalPathInsideRoot_RejectsAbsolutePathOutsideRoot(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outside, "file.txt"), []byte("x"), 0o644))

	_, err := ResolveLocalPathInsideRoot(root, filepath.Join(outside, "file.txt"))
	require.Error(t, err)
	require.Contains(t, err.Error(), ErrorPathOutsideRoot)
}
