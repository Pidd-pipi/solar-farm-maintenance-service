package main

import "errors"

type WorkOrder struct {
	ID     string `json:"id"`
	Array  string `json:"array"`
	Issue  string `json:"issue"`
	Status string `json:"status"`
}

// workStatuses lists every status a maintenance task may hold. Anything outside
// this set is rejected up front by validateWorkStatus so the store never persists
// a value that is impossible to read back from the transition table.
var workStatuses = map[string]bool{
	"open":        true,
	"scheduled":   true,
	"in_progress": true,
	"completed":   true,
}

// workTransitions is the allowed state machine for a maintenance task. A task
// that has not been scheduled yet (open) must be scheduled (or started) before
// it can be completed; it cannot jump straight to "completed". A scheduled task
// moves forward into "in_progress", which is the only path that may then close
// out as "completed".
var workTransitions = map[string]map[string]bool{
	"open":        {"scheduled": true, "in_progress": true},
	"scheduled":   {"in_progress": true},
	"in_progress": {"completed": true},
	"completed":   {},
}

// ErrWorkNotFound is returned when no maintenance task matches the given id.
var ErrWorkNotFound = errors.New("maintenance task not found")

// ErrWorkTransition is returned when the requested status change is not a legal
// step in workTransitions. Surfaces to clients as a 409 Conflict so a rejected
// update can never look like a silent success.
var ErrWorkTransition = errors.New("maintenance task status transition is not allowed")
