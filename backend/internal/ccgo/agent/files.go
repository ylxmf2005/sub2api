package agent

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/protocol"
)

type FileService struct {
	root LocalRoot
}

func NewFileService(root LocalRoot) *FileService {
	return &FileService{root: root}
}

func (s *FileService) Stat(_ context.Context, req protocol.FileStatRequest) (protocol.FileStatResponse, error) {
	path, err := s.root.ResolveProjectPath(req.Path)
	if err != nil {
		return protocol.FileStatResponse{}, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return protocol.FileStatResponse{}, localPathError(err)
	}
	rel, err := protocol.RelativePath(s.root.Path(), path)
	if err != nil {
		return protocol.FileStatResponse{}, err
	}
	return fileStatResponse(rel, info), nil
}

func (s *FileService) Read(_ context.Context, req protocol.FileReadRequest) (protocol.FileReadResponse, error) {
	if req.Offset < 0 || req.Limit < 0 {
		return protocol.FileReadResponse{}, protocol.NewError(protocol.ErrorInvalidRequest, "offset and limit must be non-negative")
	}
	path, err := s.root.ResolveProjectPath(req.Path)
	if err != nil {
		return protocol.FileReadResponse{}, err
	}
	file, err := os.Open(path)
	if err != nil {
		return protocol.FileReadResponse{}, localPathError(err)
	}
	defer file.Close()

	if req.Offset > 0 {
		if _, err := file.Seek(req.Offset, io.SeekStart); err != nil {
			return protocol.FileReadResponse{}, err
		}
	}
	limit := req.Limit
	if limit == 0 {
		limit = 4 * 1024 * 1024
	}
	data := make([]byte, limit)
	n, err := file.Read(data)
	if err != nil && err != io.EOF {
		return protocol.FileReadResponse{}, err
	}
	return protocol.FileReadResponse{Data: data[:n], EOF: err == io.EOF || int64(n) < limit}, nil
}

func (s *FileService) Write(_ context.Context, req protocol.FileWriteRequest) (protocol.FileWriteResponse, error) {
	if req.Offset < 0 {
		return protocol.FileWriteResponse{}, protocol.NewError(protocol.ErrorInvalidRequest, "offset must be non-negative")
	}
	path, err := s.root.ResolveProjectPath(req.Path)
	if err != nil {
		return protocol.FileWriteResponse{}, err
	}
	flag := os.O_CREATE | os.O_WRONLY
	if req.Truncate {
		flag |= os.O_TRUNC
	}
	file, err := os.OpenFile(path, flag, 0o644)
	if err != nil {
		return protocol.FileWriteResponse{}, localPathError(err)
	}
	defer file.Close()

	if req.Offset > 0 {
		if _, err := file.Seek(req.Offset, io.SeekStart); err != nil {
			return protocol.FileWriteResponse{}, err
		}
	}
	n, err := file.Write(req.Data)
	if err != nil {
		return protocol.FileWriteResponse{}, err
	}
	if err := file.Sync(); err != nil {
		return protocol.FileWriteResponse{}, err
	}
	return protocol.FileWriteResponse{BytesWritten: n}, nil
}

func (s *FileService) List(_ context.Context, req protocol.FileListRequest) (protocol.FileListResponse, error) {
	path, err := s.root.ResolveProjectPath(req.Path)
	if err != nil {
		return protocol.FileListResponse{}, err
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return protocol.FileListResponse{}, localPathError(err)
	}
	out := make([]protocol.FileListEntry, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return protocol.FileListResponse{}, localPathError(err)
		}
		out = append(out, protocol.FileListEntry{
			Name:    entry.Name(),
			IsDir:   info.IsDir(),
			Size:    info.Size(),
			Mode:    uint32(info.Mode()),
			ModTime: info.ModTime(),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return protocol.FileListResponse{Entries: out}, nil
}

func fileStatResponse(rel string, info os.FileInfo) protocol.FileStatResponse {
	return protocol.FileStatResponse{
		Path:    filepath.ToSlash(rel),
		IsDir:   info.IsDir(),
		Size:    info.Size(),
		Mode:    uint32(info.Mode()),
		ModTime: info.ModTime(),
	}
}
