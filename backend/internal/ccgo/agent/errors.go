package agent

import (
	"errors"
	"os"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/protocol"
)

func localPathError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, os.ErrNotExist):
		return protocol.NewError(protocol.ErrorFileNotFound, err.Error())
	case errors.Is(err, os.ErrPermission):
		return protocol.NewError(protocol.ErrorPermissionDenied, err.Error())
	default:
		return err
	}
}
