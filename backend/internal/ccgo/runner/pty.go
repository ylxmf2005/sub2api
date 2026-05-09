package runner

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

type PTY interface {
	io.ReadWriteCloser
	Resize(cols, rows uint16) error
}

type PTYStarter interface {
	Start(cmd *exec.Cmd, cols, rows uint16) (PTY, error)
}

type OSPTYStarter struct{}

func (OSPTYStarter) Start(cmd *exec.Cmd, cols, rows uint16) (PTY, error) {
	if cmd == nil {
		return nil, fmt.Errorf("ccgo command is required")
	}
	if cols == 0 {
		cols = 120
	}
	if rows == 0 {
		rows = 40
	}
	file, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: cols, Rows: rows})
	if err != nil {
		return nil, err
	}
	return osPTY{file: file}, nil
}

type osPTY struct {
	file *os.File
}

func (p osPTY) Read(data []byte) (int, error) {
	return p.file.Read(data)
}

func (p osPTY) Write(data []byte) (int, error) {
	return p.file.Write(data)
}

func (p osPTY) Close() error {
	return p.file.Close()
}

func (p osPTY) Resize(cols, rows uint16) error {
	if cols == 0 {
		cols = 120
	}
	if rows == 0 {
		rows = 40
	}
	return pty.Setsize(p.file, &pty.Winsize{Cols: cols, Rows: rows})
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
