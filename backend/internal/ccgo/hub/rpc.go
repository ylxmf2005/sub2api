package hub

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/protocol"
)

func (m *ConnectionManager) Request(ctx context.Context, workspaceID int64, method string, payload any, out any) error {
	conn, ok := m.Get(workspaceID)
	if !ok {
		return protocol.NewError(protocol.ErrorAgentDisconnected, "agent is not connected")
	}
	return conn.Request(ctx, method, payload, out)
}

func (m *ConnectionManager) FileStat(ctx context.Context, workspaceID int64, req protocol.FileStatRequest) (protocol.FileStatResponse, error) {
	var out protocol.FileStatResponse
	err := m.Request(ctx, workspaceID, protocol.MethodFileStat, req, &out)
	return out, err
}

func (m *ConnectionManager) FileRead(ctx context.Context, workspaceID int64, req protocol.FileReadRequest) (protocol.FileReadResponse, error) {
	var out protocol.FileReadResponse
	err := m.Request(ctx, workspaceID, protocol.MethodFileRead, req, &out)
	return out, err
}

func (m *ConnectionManager) FileWrite(ctx context.Context, workspaceID int64, req protocol.FileWriteRequest) (protocol.FileWriteResponse, error) {
	var out protocol.FileWriteResponse
	err := m.Request(ctx, workspaceID, protocol.MethodFileWrite, req, &out)
	return out, err
}

func (m *ConnectionManager) FileList(ctx context.Context, workspaceID int64, req protocol.FileListRequest) (protocol.FileListResponse, error) {
	var out protocol.FileListResponse
	err := m.Request(ctx, workspaceID, protocol.MethodFileList, req, &out)
	return out, err
}

func (m *ConnectionManager) FileMkdir(ctx context.Context, workspaceID int64, req protocol.FileMkdirRequest) (protocol.FileStatResponse, error) {
	var out protocol.FileStatResponse
	err := m.Request(ctx, workspaceID, protocol.MethodFileMkdir, req, &out)
	return out, err
}

func (m *ConnectionManager) FileRemove(ctx context.Context, workspaceID int64, req protocol.FileRemoveRequest) error {
	return m.Request(ctx, workspaceID, protocol.MethodFileRemove, req, nil)
}

func (m *ConnectionManager) FileRename(ctx context.Context, workspaceID int64, req protocol.FileRenameRequest) error {
	return m.Request(ctx, workspaceID, protocol.MethodFileRename, req, nil)
}

func (m *ConnectionManager) FileTruncate(ctx context.Context, workspaceID int64, req protocol.FileTruncateRequest) (protocol.FileStatResponse, error) {
	var out protocol.FileStatResponse
	err := m.Request(ctx, workspaceID, protocol.MethodFileTruncate, req, &out)
	return out, err
}

func (m *ConnectionManager) FileChmod(ctx context.Context, workspaceID int64, req protocol.FileChmodRequest) (protocol.FileStatResponse, error) {
	var out protocol.FileStatResponse
	err := m.Request(ctx, workspaceID, protocol.MethodFileChmod, req, &out)
	return out, err
}

func (m *ConnectionManager) Exec(ctx context.Context, workspaceID int64, req protocol.ExecRequest) (protocol.ExecResponse, error) {
	var out protocol.ExecResponse
	err := m.Request(ctx, workspaceID, protocol.MethodExec, req, &out)
	return out, err
}
