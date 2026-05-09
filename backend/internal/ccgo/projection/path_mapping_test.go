package projection

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJoinProjectPath(t *testing.T) {
	require.Equal(t, "file.txt", joinProjectPath(".", "file.txt"))
	require.Equal(t, "dir/file.txt", joinProjectPath("dir", "file.txt"))
	require.Equal(t, "dir/file.txt", joinProjectPath("dir/", "./file.txt"))
	require.Equal(t, ".", joinProjectPath(".", ""))
}

func TestInodeForPathStable(t *testing.T) {
	require.Equal(t, inodeForPath("dir/file.txt"), inodeForPath("dir/file.txt"))
	require.NotEqual(t, inodeForPath("dir/file.txt"), inodeForPath("dir/other.txt"))
	require.NotZero(t, inodeForPath("."))
}
