package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type PromptConfig struct {
	LocalRootDisplay string
	ServerRoot       string
	PathStyle        string
}

func GeneratePrompt(cfg PromptConfig) (string, error) {
	localRoot := strings.TrimSpace(cfg.LocalRootDisplay)
	serverRoot := strings.TrimSpace(cfg.ServerRoot)
	if localRoot == "" {
		return "", fmt.Errorf("ccgo local root is required for Claude prompt")
	}
	if serverRoot == "" {
		return "", fmt.Errorf("ccgo server root is required for Claude prompt")
	}
	pathStyle := strings.TrimSpace(cfg.PathStyle)
	if pathStyle == "" {
		pathStyle = "posix"
	}
	return fmt.Sprintf(`# ccgo Reverse Workstation

You are running inside ccgo. Treat the user's local project directory as the canonical workspace:

- Canonical local workspace: %s
- Internal server projection: %s
- Local path style: %s

The internal server projection exists only so Claude Code can attach its own project session to a stable directory. File reads and edits under the projection are served by the user's local machine. Shell commands launched through Claude Code execute on the user's local machine through ccgo, not on the server.

When referring to files, prefer project-relative paths. In user-facing answers, refer to the local workspace or relative paths; do not present the internal server projection as the real workspace. If you see both paths, the local path is the source of truth.
`, localRoot, serverRoot, pathStyle), nil
}

func WritePromptFile(dir string, cfg PromptConfig) (string, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return "", fmt.Errorf("ccgo prompt directory is required")
	}
	prompt, err := GeneratePrompt(cfg)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create ccgo prompt directory: %w", err)
	}
	path := filepath.Join(dir, "append-system-prompt.md")
	if err := os.WriteFile(path, []byte(prompt), 0o600); err != nil {
		return "", fmt.Errorf("write ccgo prompt file: %w", err)
	}
	return path, nil
}
