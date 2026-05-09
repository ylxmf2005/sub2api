package runner

import (
	"fmt"
	"io"
	"os/exec"
	"sync"
)

type Session struct {
	WorkspaceID int64
	RunID       string
	Cmd         *exec.Cmd
	PTY         PTY
	Bridge      *ExecBridge
	Mount       interface{ Unmount() error }
}

func (s *Session) Attach(input io.Reader, output io.Writer) error {
	if s == nil || s.PTY == nil {
		return fmt.Errorf("ccgo session PTY is not running")
	}
	return TerminalStream{Input: input, Output: output, PTY: s.PTY}.Attach()
}

func (s *Session) Stop() error {
	if s == nil {
		return nil
	}
	if s.Cmd != nil && s.Cmd.Process != nil {
		_ = s.Cmd.Process.Kill()
	}
	if s.PTY != nil {
		_ = s.PTY.Close()
	}
	if s.Bridge != nil {
		_ = s.Bridge.Close()
	}
	if s.Mount != nil {
		return s.Mount.Unmount()
	}
	return nil
}

type ProcessStore struct {
	mu   sync.Mutex
	runs map[int64]*Session
}

func NewProcessStore() *ProcessStore {
	return &ProcessStore{runs: make(map[int64]*Session)}
}

func (s *ProcessStore) Put(session *Session) error {
	if session == nil {
		return fmt.Errorf("ccgo session is required")
	}
	workspaceID := session.WorkspaceID
	if workspaceID <= 0 {
		return fmt.Errorf("ccgo workspace id is required")
	}
	if session.Cmd == nil || session.Cmd.Process == nil {
		return fmt.Errorf("ccgo process is not running")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.runs[workspaceID]; exists {
		return fmt.Errorf("ccgo workstation is already running for workspace %d", workspaceID)
	}
	s.runs[workspaceID] = session
	return nil
}

func (s *ProcessStore) Get(workspaceID int64) (*Session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.runs[workspaceID]
	if !ok || session == nil || session.Cmd == nil || session.Cmd.Process == nil {
		return nil, false
	}
	return session, true
}

func (s *ProcessStore) Delete(workspaceID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.runs, workspaceID)
}
