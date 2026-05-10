package projection

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/protocol"
)

type Backend interface {
	Stat(ctx context.Context, path string) (protocol.FileStatResponse, error)
	Read(ctx context.Context, path string, offset int64, limit int64) (protocol.FileReadResponse, error)
	Write(ctx context.Context, path string, data []byte, offset int64, truncate bool) (protocol.FileWriteResponse, error)
	List(ctx context.Context, path string) (protocol.FileListResponse, error)
	Mkdir(ctx context.Context, path string, mode uint32) (protocol.FileStatResponse, error)
	Remove(ctx context.Context, path string, dir bool) error
	Rename(ctx context.Context, oldPath string, newPath string) error
	Truncate(ctx context.Context, path string, size int64) (protocol.FileStatResponse, error)
	Chmod(ctx context.Context, path string, mode uint32) (protocol.FileStatResponse, error)
	Readlink(ctx context.Context, path string) (protocol.FileReadlinkResponse, error)
}

type HubBackend struct {
	workspaceID int64
	requester   interface {
		FileStat(context.Context, int64, protocol.FileStatRequest) (protocol.FileStatResponse, error)
		FileRead(context.Context, int64, protocol.FileReadRequest) (protocol.FileReadResponse, error)
		FileWrite(context.Context, int64, protocol.FileWriteRequest) (protocol.FileWriteResponse, error)
		FileList(context.Context, int64, protocol.FileListRequest) (protocol.FileListResponse, error)
		FileMkdir(context.Context, int64, protocol.FileMkdirRequest) (protocol.FileStatResponse, error)
		FileRemove(context.Context, int64, protocol.FileRemoveRequest) error
		FileRename(context.Context, int64, protocol.FileRenameRequest) error
		FileTruncate(context.Context, int64, protocol.FileTruncateRequest) (protocol.FileStatResponse, error)
		FileChmod(context.Context, int64, protocol.FileChmodRequest) (protocol.FileStatResponse, error)
		FileReadlink(context.Context, int64, protocol.FileReadlinkRequest) (protocol.FileReadlinkResponse, error)
	}
}

func NewHubBackend(workspaceID int64, requester interface {
	FileStat(context.Context, int64, protocol.FileStatRequest) (protocol.FileStatResponse, error)
	FileRead(context.Context, int64, protocol.FileReadRequest) (protocol.FileReadResponse, error)
	FileWrite(context.Context, int64, protocol.FileWriteRequest) (protocol.FileWriteResponse, error)
	FileList(context.Context, int64, protocol.FileListRequest) (protocol.FileListResponse, error)
	FileMkdir(context.Context, int64, protocol.FileMkdirRequest) (protocol.FileStatResponse, error)
	FileRemove(context.Context, int64, protocol.FileRemoveRequest) error
	FileRename(context.Context, int64, protocol.FileRenameRequest) error
	FileTruncate(context.Context, int64, protocol.FileTruncateRequest) (protocol.FileStatResponse, error)
	FileChmod(context.Context, int64, protocol.FileChmodRequest) (protocol.FileStatResponse, error)
	FileReadlink(context.Context, int64, protocol.FileReadlinkRequest) (protocol.FileReadlinkResponse, error)
}) *HubBackend {
	return &HubBackend{workspaceID: workspaceID, requester: requester}
}

func (b *HubBackend) Stat(ctx context.Context, path string) (protocol.FileStatResponse, error) {
	return b.requester.FileStat(ctx, b.workspaceID, protocol.FileStatRequest{Path: path})
}

func (b *HubBackend) Read(ctx context.Context, path string, offset int64, limit int64) (protocol.FileReadResponse, error) {
	return b.requester.FileRead(ctx, b.workspaceID, protocol.FileReadRequest{Path: path, Offset: offset, Limit: limit})
}

func (b *HubBackend) Write(ctx context.Context, path string, data []byte, offset int64, truncate bool) (protocol.FileWriteResponse, error) {
	return b.requester.FileWrite(ctx, b.workspaceID, protocol.FileWriteRequest{Path: path, Data: data, Offset: offset, Truncate: truncate})
}

func (b *HubBackend) List(ctx context.Context, path string) (protocol.FileListResponse, error) {
	return b.requester.FileList(ctx, b.workspaceID, protocol.FileListRequest{Path: path})
}

func (b *HubBackend) Mkdir(ctx context.Context, path string, mode uint32) (protocol.FileStatResponse, error) {
	return b.requester.FileMkdir(ctx, b.workspaceID, protocol.FileMkdirRequest{Path: path, Mode: mode})
}

func (b *HubBackend) Remove(ctx context.Context, path string, dir bool) error {
	return b.requester.FileRemove(ctx, b.workspaceID, protocol.FileRemoveRequest{Path: path, Dir: dir})
}

func (b *HubBackend) Rename(ctx context.Context, oldPath string, newPath string) error {
	return b.requester.FileRename(ctx, b.workspaceID, protocol.FileRenameRequest{OldPath: oldPath, NewPath: newPath})
}

func (b *HubBackend) Truncate(ctx context.Context, path string, size int64) (protocol.FileStatResponse, error) {
	return b.requester.FileTruncate(ctx, b.workspaceID, protocol.FileTruncateRequest{Path: path, Size: size})
}

func (b *HubBackend) Chmod(ctx context.Context, path string, mode uint32) (protocol.FileStatResponse, error) {
	return b.requester.FileChmod(ctx, b.workspaceID, protocol.FileChmodRequest{Path: path, Mode: mode})
}

func (b *HubBackend) Readlink(ctx context.Context, path string) (protocol.FileReadlinkResponse, error) {
	return b.requester.FileReadlink(ctx, b.workspaceID, protocol.FileReadlinkRequest{Path: path})
}
