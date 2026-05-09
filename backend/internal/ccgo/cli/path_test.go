package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveLocalPath_CanonicalizesSymlinkedDirectory(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project")
	require.NoError(t, os.Mkdir(project, 0o755))
	link := filepath.Join(root, "link")
	if runtime.GOOS != "windows" {
		require.NoError(t, os.Symlink(project, link))
	} else {
		link = project
	}

	info, err := ResolveLocalPath(link)
	require.NoError(t, err)
	expected, err := filepath.EvalSymlinks(project)
	require.NoError(t, err)
	require.Equal(t, expected, info.CanonicalRoot)
	require.Equal(t, expected, info.LocalRootDisplay)
	require.Len(t, info.LocalRootHash, 64)
	require.NotEmpty(t, info.OS)
	require.NotEmpty(t, info.PathStyle)
}

func TestResolveLocalPath_RejectsFile(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "file.txt")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0o644))

	_, err := ResolveLocalPath(file)
	require.ErrorContains(t, err, "directory")
}
