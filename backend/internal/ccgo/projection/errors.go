package projection

import (
	"errors"
	"syscall"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/protocol"
)

func errnoFromError(err error) syscall.Errno {
	if err == nil {
		return 0
	}
	var ccgoErr *protocol.Error
	if errors.As(err, &ccgoErr) {
		switch ccgoErr.Code {
		case protocol.ErrorAgentDisconnected, protocol.ErrorRequestTimeout:
			return syscall.EIO
		case protocol.ErrorFileNotFound:
			return syscall.ENOENT
		case protocol.ErrorPermissionDenied:
			return syscall.EACCES
		case protocol.ErrorInvalidPath, protocol.ErrorPathOutsideRoot, protocol.ErrorInvalidRequest:
			return syscall.EINVAL
		case protocol.ErrorUnsupported:
			return syscall.ENOTSUP
		}
	}
	return syscall.EIO
}
