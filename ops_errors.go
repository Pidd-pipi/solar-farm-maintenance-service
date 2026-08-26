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

func wrapOps(code, operation string, cause error) error {
	return fmt.Errorf("%s: %s: %v", code, operation, cause)
}

// opsCode classifies an error into a stable coarse code used for HTTP status
// mapping. Sentinel errors take precedence so wrapped domain errors keep their
// identity even when they pass through several service layers.
func opsCode(err error) string {
	var typed *OpsError
	if errors.As(err, &typed) {
		return typed.Code
	}
	return "internal"
}

func opsIsNotFound(err error) bool   { return errors.Is(err, ErrOpsNotFound) }
func opsIsConflict(err error) bool   { return errors.Is(err, ErrOpsConflict) }
func opsIsInvalid(err error) bool    { return errors.Is(err, ErrOpsInvalid) }
func opsIsTransition(err error) bool { return errors.Is(err, ErrOpsTransition) }
func opsIsPolicy(err error) bool     { return errors.Is(err, ErrOpsPolicy) }
