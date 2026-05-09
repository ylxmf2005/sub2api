package protocol

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	MessageTypeRequest   = "request"
	MessageTypeResponse  = "response"
	MessageTypeHeartbeat = "heartbeat"
	MessageTypeTerminalInput = "terminal.input"
	MessageTypeTerminalOutput = "terminal.output"

	MethodFileStat     = "file.stat"
	MethodFileRead     = "file.read"
	MethodFileWrite    = "file.write"
	MethodFileList     = "file.list"
	MethodFileMkdir    = "file.mkdir"
	MethodFileRemove   = "file.remove"
	MethodFileRename   = "file.rename"
	MethodFileTruncate = "file.truncate"
	MethodFileChmod    = "file.chmod"
	MethodExec         = "exec"
)

type Envelope struct {
	Type      string          `json:"type"`
	RequestID string          `json:"request_id,omitempty"`
	Method    string          `json:"method,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Error     *Error          `json:"error,omitempty"`
}

type FileStatRequest struct {
	Path string `json:"path"`
}

type FileStatResponse struct {
	Path    string    `json:"path"`
	IsDir   bool      `json:"is_dir"`
	Size    int64     `json:"size"`
	Mode    uint32    `json:"mode"`
	ModTime time.Time `json:"mod_time"`
}

type FileReadRequest struct {
	Path   string `json:"path"`
	Offset int64  `json:"offset"`
	Limit  int64  `json:"limit"`
}

type FileReadResponse struct {
	Data []byte `json:"data"`
	EOF  bool   `json:"eof"`
}

type FileWriteRequest struct {
	Path     string `json:"path"`
	Data     []byte `json:"data"`
	Offset   int64  `json:"offset"`
	Truncate bool   `json:"truncate"`
}

type FileWriteResponse struct {
	BytesWritten int `json:"bytes_written"`
}

type FileListRequest struct {
	Path string `json:"path"`
}

type FileListEntry struct {
	Name    string    `json:"name"`
	IsDir   bool      `json:"is_dir"`
	Size    int64     `json:"size"`
	Mode    uint32    `json:"mode"`
	ModTime time.Time `json:"mod_time"`
}

type FileListResponse struct {
	Entries []FileListEntry `json:"entries"`
}

type FileMkdirRequest struct {
	Path string `json:"path"`
	Mode uint32 `json:"mode"`
}

type FileRemoveRequest struct {
	Path string `json:"path"`
	Dir  bool   `json:"dir"`
}

type FileRenameRequest struct {
	OldPath string `json:"old_path"`
	NewPath string `json:"new_path"`
}

type FileTruncateRequest struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

type FileChmodRequest struct {
	Path string `json:"path"`
	Mode uint32 `json:"mode"`
}

type ExecRequest struct {
	Cwd       string            `json:"cwd"`
	Command   string            `json:"command"`
	Env       map[string]string `json:"env,omitempty"`
	TimeoutMS int64             `json:"timeout_ms,omitempty"`
}

type ExecResponse struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}

func NewRequest(requestID, method string, payload any) (Envelope, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return Envelope{}, fmt.Errorf("encode ccgo request payload: %w", err)
	}
	return Envelope{Type: MessageTypeRequest, RequestID: requestID, Method: method, Payload: raw}, nil
}

func NewResponse(requestID string, payload any) (Envelope, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return Envelope{}, fmt.Errorf("encode ccgo response payload: %w", err)
	}
	return Envelope{Type: MessageTypeResponse, RequestID: requestID, Payload: raw}, nil
}

func NewErrorResponse(requestID string, err *Error) Envelope {
	return Envelope{Type: MessageTypeResponse, RequestID: requestID, Error: err}
}

func DecodePayload[T any](env Envelope) (T, error) {
	var out T
	if len(env.Payload) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(env.Payload, &out); err != nil {
		return out, fmt.Errorf("decode ccgo payload: %w", err)
	}
	return out, nil
}
