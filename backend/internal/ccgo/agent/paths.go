package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/protocol"
)

type LocalRoot struct {
	path string
}

func NewLocalRoot(root string) (LocalRoot, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return LocalRoot{}, protocol.NewError(protocol.ErrorInvalidPath, "local root is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return LocalRoot{}, fmt.Errorf("resolve local root: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return LocalRoot{}, localPathError(err)
	}
	if !info.IsDir() {
		return LocalRoot{}, protocol.NewError(protocol.ErrorInvalidPath, "local root must be a directory")
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return LocalRoot{}, localPathError(err)
	}
	return LocalRoot{path: resolved}, nil
}

func (r LocalRoot) Path() string {
	return r.path
}

func (r LocalRoot) ResolveProjectPath(requested string) (string, error) {
	if r.path == "" {
		return "", protocol.NewError(protocol.ErrorInvalidPath, "local root is not configured")
	}
	if runtime.GOOS == "windows" && looksLikeWindowsDriveEscape(requested) {
		return "", protocol.NewError(protocol.ErrorPathOutsideRoot, "path escapes the ccgo workspace volume")
	}
	return protocol.ResolveInsideRoot(r.path, requested)
}

func (r LocalRoot) ResolveProjectPathNoFollow(requested string) (string, error) {
	if r.path == "" {
		return "", protocol.NewError(protocol.ErrorInvalidPath, "local root is not configured")
	}
	if runtime.GOOS == "windows" && looksLikeWindowsDriveEscape(requested) {
		return "", protocol.NewError(protocol.ErrorPathOutsideRoot, "path escapes the ccgo workspace volume")
	}
	return protocol.ResolveInsideRootNoFollow(r.path, requested)
}

func (r LocalRoot) ResolveLocalCwd(cwd string) (string, error) {
	if r.path == "" {
		return "", protocol.NewError(protocol.ErrorInvalidPath, "local root is not configured")
	}
	if runtime.GOOS == "windows" && looksLikeWindowsDriveEscape(cwd) {
		return "", protocol.NewError(protocol.ErrorPathOutsideRoot, "path escapes the ccgo workspace volume")
	}
	return protocol.ResolveLocalPathInsideRoot(r.path, cwd)
}

func looksLikeWindowsDriveEscape(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) < 2 {
		return false
	}
	if value[1] != ':' {
		return false
	}
	first := value[0]
	return (first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z')
}
