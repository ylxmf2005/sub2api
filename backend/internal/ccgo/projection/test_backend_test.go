package projection

import (
	"context"
	"sort"
	"syscall"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/agent"
	"github.com/Wei-Shaw/sub2api/internal/ccgo/protocol"
	"github.com/hanwen/go-fuse/v2/fs"
)

type localBackend struct {
	files *agent.FileService
}

func newLocalBackend(root string) (*localBackend, error) {
	localRoot, err := agent.NewLocalRoot(root)
	if err != nil {
		return nil, err
	}
	return &localBackend{files: agent.NewFileService(localRoot)}, nil
}

func (b *localBackend) Stat(ctx context.Context, path string) (protocol.FileStatResponse, error) {
	return b.files.Stat(ctx, protocol.FileStatRequest{Path: path})
}

func (b *localBackend) Read(ctx context.Context, path string, offset int64, limit int64) (protocol.FileReadResponse, error) {
	return b.files.Read(ctx, protocol.FileReadRequest{Path: path, Offset: offset, Limit: limit})
}

func (b *localBackend) Write(ctx context.Context, path string, data []byte, offset int64, truncate bool) (protocol.FileWriteResponse, error) {
	return b.files.Write(ctx, protocol.FileWriteRequest{Path: path, Data: data, Offset: offset, Truncate: truncate})
}

func (b *localBackend) List(ctx context.Context, path string) (protocol.FileListResponse, error) {
	return b.files.List(ctx, protocol.FileListRequest{Path: path})
}

func (b *localBackend) Mkdir(ctx context.Context, path string, mode uint32) (protocol.FileStatResponse, error) {
	return b.files.Mkdir(ctx, protocol.FileMkdirRequest{Path: path, Mode: mode})
}

func (b *localBackend) Remove(ctx context.Context, path string, dir bool) error {
	return b.files.Remove(ctx, protocol.FileRemoveRequest{Path: path, Dir: dir})
}

func (b *localBackend) Rename(ctx context.Context, oldPath string, newPath string) error {
	return b.files.Rename(ctx, protocol.FileRenameRequest{OldPath: oldPath, NewPath: newPath})
}

func (b *localBackend) Truncate(ctx context.Context, path string, size int64) (protocol.FileStatResponse, error) {
	return b.files.Truncate(ctx, protocol.FileTruncateRequest{Path: path, Size: size})
}

func (b *localBackend) Chmod(ctx context.Context, path string, mode uint32) (protocol.FileStatResponse, error) {
	return b.files.Chmod(ctx, protocol.FileChmodRequest{Path: path, Mode: mode})
}

func (b *localBackend) Readlink(ctx context.Context, path string) (protocol.FileReadlinkResponse, error) {
	return b.files.Readlink(ctx, protocol.FileReadlinkRequest{Path: path})
}

func readAllDirStreamNames(stream fs.DirStream) ([]string, syscall.Errno) {
	names := make([]string, 0)
	defer stream.Close()
	for stream.HasNext() {
		entry, errno := stream.Next()
		if errno != 0 {
			return nil, errno
		}
		names = append(names, entry.Name)
	}
	sort.Strings(names)
	return names, 0
}
