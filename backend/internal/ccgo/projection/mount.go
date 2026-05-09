package projection

import (
	"fmt"
	"os"
	"strings"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

type Mount struct {
	server *fuse.Server
	dir    string
}

func MountWorkspace(dir string, backend Backend) (*Mount, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil, fmt.Errorf("ccgo projection mount dir is required")
	}
	if backend == nil {
		return nil, fmt.Errorf("ccgo projection backend is required")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create ccgo projection mount dir: %w", err)
	}
	server, err := fs.Mount(dir, NewRootNode(backend), &fs.Options{
		MountOptions: fuse.MountOptions{
			Name: "ccgo",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("mount ccgo projection: %w", err)
	}
	return &Mount{server: server, dir: dir}, nil
}

func (m *Mount) Dir() string {
	if m == nil {
		return ""
	}
	return m.dir
}

func (m *Mount) Unmount() error {
	if m == nil || m.server == nil {
		return nil
	}
	return m.server.Unmount()
}
