package main

import "fmt"

func validateWorkStatus(status string) error {
	for _, allowed := range []string{"scheduled", "in_progress", "completed"} {
		if status == allowed {
			return nil
		}
	}
	return fmt.Errorf("unsupported status %q", status)
}
