package protocol

const (
	ErrorAgentDisconnected = "AGENT_DISCONNECTED"
	ErrorRequestTimeout    = "REQUEST_TIMEOUT"
	ErrorInvalidPath       = "INVALID_PATH"
	ErrorPathOutsideRoot   = "PATH_OUTSIDE_ROOT"
	ErrorFileNotFound      = "FILE_NOT_FOUND"
	ErrorPermissionDenied  = "PERMISSION_DENIED"
	ErrorInvalidRequest    = "INVALID_REQUEST"
	ErrorUnsupported       = "UNSUPPORTED"
	ErrorInternal          = "INTERNAL"
)

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Message != "" {
		return e.Code + ": " + e.Message
	}
	return e.Code
}

func NewError(code, message string) *Error {
	return &Error{Code: code, Message: message}
}
