package service

import (
	"context"
	"sync"

	"iptv-tool-v2/pkg/utils"
)

// CrackState represents the current state of the crack task
type CrackState string

const (
	CrackStateIdle      CrackState = "idle"
	CrackStateRunning   CrackState = "running"
	CrackStateCompleted CrackState = "completed"
	CrackStateFailed    CrackState = "failed"
	CrackStateStopped   CrackState = "stopped"
)

// CrackTaskStatus holds the full status of the crack task
type CrackTaskStatus struct {
	State         CrackState           `json:"state"`
	Authenticator string               `json:"authenticator,omitempty"`
	Mode          utils.CrackMode      `json:"mode,omitempty"`
	Progress      *utils.CrackProgress `json:"progress,omitempty"`
	Result        *utils.CrackResult   `json:"result,omitempty"`
	Error         string               `json:"error,omitempty"`
}

// CrackTaskManager manages a singleton background crack task
type CrackTaskManager struct {
	mu            sync.Mutex
	state         CrackState
	authenticator string
	mode          utils.CrackMode
	progress      *utils.CrackProgress
	result        *utils.CrackResult
	errorMsg      string
	cancel        context.CancelFunc
}

// global singleton
var crackManager = &CrackTaskManager{
	state: CrackStateIdle,
}

// GetCrackManager returns the global crack task manager
func GetCrackManager() *CrackTaskManager {
	return crackManager
}

// Start begins a new crack task in the background.
// Returns an error message if a task is already running.
func (m *CrackTaskManager) Start(authenticator string, mode utils.CrackMode) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state == CrackStateRunning {
		return "crack task is already running"
	}

	// Reset state
	m.state = CrackStateRunning
	m.authenticator = authenticator
	m.mode = mode
	m.progress = nil
	m.result = nil
	m.errorMsg = ""

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	go func() {
		progressCb := func(p utils.CrackProgress) {
			m.mu.Lock()
			defer m.mu.Unlock()
			if m.state == CrackStateRunning {
				pCopy := p
				m.progress = &pCopy
			}
		}

		result, err := utils.CrackAuthenticator(ctx, authenticator, mode, progressCb)

		m.mu.Lock()
		defer m.mu.Unlock()

		// Only update if we're still in running state (not stopped by user)
		if m.state != CrackStateRunning {
			return
		}

		if err != nil {
			if ctx.Err() != nil {
				m.state = CrackStateStopped
			} else {
				m.state = CrackStateFailed
				m.errorMsg = err.Error()
			}
		} else {
			m.state = CrackStateCompleted
			m.result = result
		}
	}()

	return ""
}

// Stop cancels the current running crack task
func (m *CrackTaskManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state == CrackStateRunning && m.cancel != nil {
		m.cancel()
		m.state = CrackStateStopped
		m.cancel = nil
	}
}

// GetStatus returns the current status of the crack task
func (m *CrackTaskManager) GetStatus() CrackTaskStatus {
	m.mu.Lock()
	defer m.mu.Unlock()

	status := CrackTaskStatus{
		State: m.state,
	}

	if m.state != CrackStateIdle {
		status.Authenticator = m.authenticator
		status.Mode = m.mode
	}

	if m.progress != nil {
		pCopy := *m.progress
		status.Progress = &pCopy
	}

	if m.result != nil {
		status.Result = m.result
	}

	if m.errorMsg != "" {
		status.Error = m.errorMsg
	}

	return status
}
