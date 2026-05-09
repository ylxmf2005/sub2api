package runner

import (
	"fmt"
	"io"
)

type PTY interface {
	io.ReadWriteCloser
	Resize(cols, rows uint16) error
}

type UnsupportedPTY struct{}

func (UnsupportedPTY) Read([]byte) (int, error) {
	return 0, fmt.Errorf("ccgo server PTY is not initialized")
}

func (UnsupportedPTY) Write([]byte) (int, error) {
	return 0, fmt.Errorf("ccgo server PTY is not initialized")
}

func (UnsupportedPTY) Close() error {
	return nil
}

func (UnsupportedPTY) Resize(uint16, uint16) error {
	return fmt.Errorf("ccgo server PTY is not initialized")
}
