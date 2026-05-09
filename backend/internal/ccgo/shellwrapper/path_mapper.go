package shellwrapper

import (
	"fmt"
	"path"
	"strings"
	"unicode"
)

type PathMapper struct {
	ServerRoot string
	LocalRoot  string
}

func NewPathMapper(serverRoot, localRoot string) (*PathMapper, error) {
	serverRoot = cleanServerRoot(serverRoot)
	localRoot = strings.TrimRight(strings.TrimSpace(localRoot), `/\`)
	if serverRoot == "" {
		return nil, fmt.Errorf("ccgo server root is required")
	}
	if localRoot == "" {
		return nil, fmt.Errorf("ccgo local root is required")
	}
	if !strings.HasPrefix(serverRoot, "/") {
		return nil, fmt.Errorf("ccgo server root must be an absolute projection path")
	}
	return &PathMapper{ServerRoot: serverRoot, LocalRoot: localRoot}, nil
}

func (m *PathMapper) MapCwd(serverCwd string) (string, error) {
	serverCwd = cleanServerRoot(serverCwd)
	if serverCwd == "" {
		serverCwd = m.ServerRoot
	}
	if serverCwd == m.ServerRoot {
		return m.LocalRoot, nil
	}
	if !strings.HasPrefix(serverCwd, m.ServerRoot+"/") {
		return "", fmt.Errorf("ccgo cwd %q is outside projection root %q", serverCwd, m.ServerRoot)
	}
	return joinLocalPath(m.LocalRoot, strings.TrimPrefix(serverCwd, m.ServerRoot+"/")), nil
}

func (m *PathMapper) MapCommand(command string) string {
	if command == "" || m == nil || m.ServerRoot == "" || m.LocalRoot == "" {
		return command
	}
	var out strings.Builder
	out.Grow(len(command) + 16)
	quote := byte(0)
	for i := 0; i < len(command); {
		if quote == 0 && (command[i] == '\'' || command[i] == '"') {
			quote = command[i]
			out.WriteByte(command[i])
			i++
			continue
		}
		if quote == '"' && command[i] == '\\' && i+1 < len(command) {
			out.WriteByte(command[i])
			out.WriteByte(command[i+1])
			i += 2
			continue
		}
		if quote != 0 && command[i] == quote {
			quote = 0
			out.WriteByte(command[i])
			i++
			continue
		}
		if pathStartBoundary(command, i) && strings.HasPrefix(command[i:], m.ServerRoot) && pathEndBoundary(command, i+len(m.ServerRoot)) {
			end := pathEnd(command, i, quote)
			mapped := joinLocalPath(m.LocalRoot, m.serverPathSuffix(command[i:end]))
			if quote == '\'' {
				out.WriteString(strings.ReplaceAll(mapped, `'`, `'\''`))
			} else if quote == '"' {
				out.WriteString(escapeDoubleQuoted(mapped))
			} else {
				out.WriteString(shellQuote(mapped))
			}
			i = end
			continue
		}
		out.WriteByte(command[i])
		i++
	}
	return out.String()
}

func (m *PathMapper) serverPathSuffix(value string) string {
	value = cleanServerRoot(value)
	if value == m.ServerRoot {
		return ""
	}
	return strings.TrimPrefix(value, m.ServerRoot+"/")
}

func cleanServerRoot(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	cleaned := path.Clean(strings.ReplaceAll(value, "\\", "/"))
	if cleaned == "." {
		return ""
	}
	return cleaned
}

func pathStartBoundary(command string, idx int) bool {
	if idx == 0 {
		return true
	}
	prev := command[idx-1]
	if prev == '\'' || prev == '"' {
		return true
	}
	return unicode.IsSpace(rune(prev)) || strings.ContainsRune("=;|&(<>{}[", rune(prev))
}

func pathEndBoundary(command string, idx int) bool {
	if idx >= len(command) {
		return true
	}
	switch command[idx] {
	case '/', '\'', '"', '\\':
		return true
	default:
		return unicode.IsSpace(rune(command[idx])) || strings.ContainsRune(";|&)<>{}[]", rune(command[idx]))
	}
}

func pathEnd(command string, start int, quote byte) int {
	for i := start; i < len(command); i++ {
		if quote == '\'' {
			if command[i] == '\'' {
				return i
			}
			continue
		}
		if quote == '"' {
			if command[i] == '\\' && i+1 < len(command) {
				i++
				continue
			}
			if command[i] == '"' {
				return i
			}
			continue
		}
		if command[i] == '\\' && i+1 < len(command) {
			i++
			continue
		}
		if unicode.IsSpace(rune(command[i])) || strings.ContainsRune(";|&)<>{}[]'\"", rune(command[i])) {
			return i
		}
	}
	return len(command)
}

func joinLocalPath(root, suffix string) string {
	suffix = strings.TrimPrefix(strings.ReplaceAll(suffix, "\\", "/"), "/")
	if suffix == "" || suffix == "." {
		return root
	}
	if strings.Contains(root, "\\") {
		return strings.TrimRight(root, `\/`) + `\` + strings.ReplaceAll(suffix, "/", `\`)
	}
	return strings.TrimRight(root, "/") + "/" + suffix
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func escapeDoubleQuoted(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`, `$`, `\$`, "`", "\\`")
	return replacer.Replace(value)
}
