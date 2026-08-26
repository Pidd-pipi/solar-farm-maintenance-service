package main

import (
	"errors"
	"fmt"
)

var (
	ErrOpsNotFound   = errors.New("operations record not found")
	ErrOpsConflict   = errors.New("operations revision conflict")
	ErrOpsInvalid    = errors.New("operations request is invalid")
	ErrOpsTransition = errors.New("operations status transition is not allowed")
	ErrOpsPolicy     = errors.New("operations policy rejected the request")
)

// OpsError carries an operation context plus an underlying cause. Unwrap keeps
// the cause on the error chain so errors.Is/As stay usable.
type OpsError struct {
	Code      string
	Operation string
	Cause     error
}

func (e *OpsError) Error() string {
	if e.Cause == nil {
		return e.Code + ": " + e.Operation
	}
	return fmt.Sprintf("%s: %s: %v", e.Code, e.Operation, e.Cause)
}
func (e *OpsError) Unwrap() error { return e.Cause }

// wrapOps attaches a stable coarse code and an operation name to cause. The
// returned error keeps cause on its chain via Unwrap so errors.Is still works.
func wrapOps(code, operation string, cause error) error {
	return &OpsError{Code: classifyOps(cause, code), Operation: operation, Cause: cause}
}

// classifyOps resolves the stable coarse code for an error. Sentinel errors are
// matched via errors.Is so wrapped domain errors keep their identity even when
// they pass through several service layers. An explicit caller hint (fallback)
// is only honoured when it names a known stable code; arbitrary hints collapse
// to "internal" so log/metric buckets cannot drift.
func classifyOps(err error, fallback string) string {
	switch {
	case errors.Is(err, ErrOpsNotFound):
		return "not_found"
	case errors.Is(err, ErrOpsConflict):
		return "conflict"
	case errors.Is(err, ErrOpsInvalid):
		return "invalid"
	case errors.Is(err, ErrOpsTransition):
		return "transition"
	case errors.Is(err, ErrOpsPolicy):
		return "policy"
	}
	switch fallback {
	case "not_found", "conflict", "invalid", "transition", "policy":
		return fallback
	}
	return "internal"
}

// opsCode classifies an error into a stable coarse code used for HTTP status
// mapping. Wrapped OpsError values expose their Code directly; otherwise the
// sentinel chain is inspected so plain fmt.Errorf("%w: ...") wrappers still map
// to the right bucket instead of collapsing to "internal".
func opsCode(err error) string {
	if err == nil {
		return ""
	}
	var typed *OpsError
	if errors.As(err, &typed) && typed.Code != "" {
		return typed.Code
	}
	return classifyOps(err, "")
}

func opsIsNotFound(err error) bool   { return errors.Is(err, ErrOpsNotFound) }
func opsIsConflict(err error) bool   { return errors.Is(err, ErrOpsConflict) }
func opsIsInvalid(err error) bool    { return errors.Is(err, ErrOpsInvalid) }
func opsIsTransition(err error) bool { return errors.Is(err, ErrOpsTransition) }
func opsIsPolicy(err error) bool     { return errors.Is(err, ErrOpsPolicy) }
