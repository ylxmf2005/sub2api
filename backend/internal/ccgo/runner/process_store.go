package runner

import (
	"fmt"
	"os/exec"
	"sync"
)

type ProcessStore struct {
	mu   sync.Mutex
	runs map[int64]*exec.Cmd
}

func NewProcessStore() *ProcessStore {
	return &ProcessStore{runs: make(map[int64]*exec.Cmd)}
}

func (s *ProcessStore) Put(workspaceID int64, cmd *exec.Cmd) error {
	if workspaceID <= 0 {
		return fmt.Errorf("ccgo workspace id is required")
	}
	if cmd == nil || cmd.Process == nil {
		return fmt.Errorf("ccgo process is not running")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.runs[workspaceID]; exists {
		return fmt.Errorf("ccgo workstation is already running for workspace %d", workspaceID)
	}
	s.runs[workspaceID] = cmd
	return nil
}

func (s *ProcessStore) Get(workspaceID int64) (*exec.Cmd, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cmd, ok := s.runs[workspaceID]
	if !ok || cmd == nil || cmd.Process == nil {
		return nil, false
	}
	return cmd, true
}

func (s *ProcessStore) Delete(workspaceID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.runs, workspaceID)
}
