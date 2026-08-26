package main

import (
	"fmt"
	"sync"
)

var opsTransitionTable = map[OpsStatus]map[OpsStatus]bool{
	OpsStatusQueued: {OpsStatusActive: true, OpsStatusClosed: true},
	OpsStatusActive: {OpsStatusPaused: true, OpsStatusClosed: true},
	OpsStatusPaused: {OpsStatusActive: true, OpsStatusClosed: true},
	OpsStatusClosed: {},
}

type OpsTransition struct {
	From   OpsStatus
	To     OpsStatus
	Reason string
}

// OpsStateMachine records a bounded transition history. TrimHistory and Reset
// reclaim backing storage so the history does not grow without bound.
type OpsStateMachine struct {
	mu      sync.RWMutex
	history []OpsTransition
}

func newOpsStateMachine() *OpsStateMachine { return &OpsStateMachine{history: []OpsTransition{}} }
func (m *OpsStateMachine) CanMove(from, to OpsStatus) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return from == to || opsTransitionTable[from][to]
}
func (m *OpsStateMachine) Move(from, to OpsStatus, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if from == to {
		return nil
	}
	if !opsTransitionTable[from][to] {
		return fmt.Errorf("%w: %s to %s", ErrOpsTransition, from, to)
	}
	m.history = append(m.history, OpsTransition{From: from, To: to, Reason: reason})
	return nil
}
func (m *OpsStateMachine) History() []OpsTransition {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]OpsTransition(nil), m.history...)
}
func (m *OpsStateMachine) Last() (OpsTransition, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.history) == 0 {
		return OpsTransition{}, false
	}
	return m.history[len(m.history)-1], true
}

// TrimHistory keeps only the most recent max transitions.
func (m *OpsStateMachine) TrimHistory(max int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if max < 0 {
		max = 0
	}
	if len(m.history) <= max {
		return
	}
	kept := append([]OpsTransition(nil), m.history[len(m.history)-max:]...)
	m.history = kept
}

// Capacity reports the current backing capacity of the history list.
func (m *OpsStateMachine) Capacity() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return cap(m.history)
}

// Reset drops the history and releases the backing storage.
func (m *OpsStateMachine) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.history = []OpsTransition{}
}

func opsStatusValid(value OpsStatus) bool {
	return value == OpsStatusQueued || value == OpsStatusActive || value == OpsStatusPaused || value == OpsStatusClosed
}
func opsStatusTerminal(value OpsStatus) bool { return value == OpsStatusClosed }
