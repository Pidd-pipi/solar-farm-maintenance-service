package main

type WorkOrder struct {
	ID     string `json:"id"`
	Array  string `json:"array"`
	Issue  string `json:"issue"`
	Status string `json:"status"`
}

var workTransitions = map[string]map[string]bool{
	"open":        {"scheduled": true},
	"scheduled":   {"in_progress": true},
	"in_progress": {"completed": true},
	"completed":   {},
}
