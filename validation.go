package main

import "fmt"

// validateWorkStatus rejects any status value that is not one of the known
// maintenance-task statuses. Stopping bad values here keeps illegal transitions
// from reaching the store, so the list returned by the store always matches what
// callers believe they set.
func validateWorkStatus(status string) error {
	if !workStatuses[status] {
		return fmt.Errorf("unknown status %q", status)
	}
	return nil
}
