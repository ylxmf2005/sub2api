package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type LocalPathInfo struct {
	CanonicalRoot    string `json:"canonical_root"`
	LocalRootHash    string `json:"local_root_hash"`
	LocalRootDisplay string `json:"local_root_display"`
	OS               string `json:"os"`
	PathStyle        string `json:"path_style"`
}

func ResolveLocalPath(input string) (*LocalPathInfo, error) {
	if strings.TrimSpace(input) == "" {
		return nil, fmt.Errorf("local path is required")
	}
	abs, err := filepath.Abs(input)
	if err != nil {
		return nil, fmt.Errorf("resolve absolute path: %w", err)
	}
	realPath, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, fmt.Errorf("resolve canonical path: %w", err)
	}
	info, err := os.Stat(realPath)
	if err != nil {
		return nil, fmt.Errorf("stat local path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("local path must be a directory")
	}
	canonical := filepath.Clean(realPath)
	pathStyle := "posix"
	if runtime.GOOS == "windows" {
		pathStyle = "windows"
	}
	hash := sha256.Sum256([]byte(runtime.GOOS + "\x00" + pathStyle + "\x00" + canonical))
	return &LocalPathInfo{
		CanonicalRoot:    canonical,
		LocalRootHash:    hex.EncodeToString(hash[:]),
		LocalRootDisplay: canonical,
		OS:               runtime.GOOS,
		PathStyle:        pathStyle,
	}, nil
}
