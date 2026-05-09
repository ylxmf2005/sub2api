package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/protocol"
	"github.com/stretchr/testify/require"
)

func TestExecServiceRunsCommandInsideRoot(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "file.txt"), []byte("hello"), 0o644))
	localRoot, err := NewLocalRoot(root)
	require.NoError(t, err)
	execSvc := NewExecService(localRoot)

	resp, err := execSvc.Run(context.Background(), protocol.ExecRequest{
		Cwd:     root,
		Command: "pwd; cat file.txt",
	})
	require.NoError(t, err)
	require.Equal(t, 0, resp.ExitCode)
	require.Contains(t, resp.Stdout, localRoot.Path())
	require.Contains(t, resp.Stdout, "hello")
}

func TestExecServicePreservesNonZeroExitAndStderr(t *testing.T) {
	root := t.TempDir()
	localRoot, err := NewLocalRoot(root)
	require.NoError(t, err)
	execSvc := NewExecService(localRoot)

	resp, err := execSvc.Run(context.Background(), protocol.ExecRequest{
		Cwd:     root,
		Command: "echo problem >&2; exit 7",
	})
	require.NoError(t, err)
	require.Equal(t, 7, resp.ExitCode)
	require.Contains(t, resp.Stderr, "problem")
}

func TestExecServiceRejectsCwdOutsideRoot(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	localRoot, err := NewLocalRoot(root)
	require.NoError(t, err)
	execSvc := NewExecService(localRoot)

	_, err = execSvc.Run(context.Background(), protocol.ExecRequest{
		Cwd:     outside,
		Command: "pwd",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), protocol.ErrorPathOutsideRoot)
}

func TestExecServiceRejectsCwdSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	require.NoError(t, os.Symlink(outside, filepath.Join(root, "link")))
	localRoot, err := NewLocalRoot(root)
	require.NoError(t, err)
	execSvc := NewExecService(localRoot)

	_, err = execSvc.Run(context.Background(), protocol.ExecRequest{
		Cwd:     filepath.Join(root, "link"),
		Command: "pwd",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), protocol.ErrorPathOutsideRoot)
}

func TestExecServiceTimesOut(t *testing.T) {
	root := t.TempDir()
	localRoot, err := NewLocalRoot(root)
	require.NoError(t, err)
	execSvc := NewExecService(localRoot)

	resp, err := execSvc.Run(context.Background(), protocol.ExecRequest{
		Cwd:       root,
		Command:   "sleep 1",
		TimeoutMS: int64((10 * time.Millisecond) / time.Millisecond),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), protocol.ErrorRequestTimeout)
	require.NotEqual(t, 0, resp.ExitCode)
}
