package runner

import (
	"io"
	"sync"
)

type TerminalStream struct {
	Input  io.Reader
	Output io.Writer
	PTY    io.ReadWriter
}

func (s TerminalStream) Attach() error {
	var wg sync.WaitGroup
	errCh := make(chan error, 2)
	copyOne := func(dst io.Writer, src io.Reader) {
		defer wg.Done()
		if dst == nil || src == nil {
			return
		}
		_, err := io.Copy(dst, src)
		if err != nil {
			errCh <- err
		}
	}
	wg.Add(2)
	go copyOne(s.PTY, s.Input)
	go copyOne(s.Output, s.PTY)
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case err := <-errCh:
		return err
	case <-done:
		return nil
	}
}
