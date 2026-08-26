package main

import (
	"fmt"
	"strings"
)

// PlanEntry is one maintenance job derived from an open operations record.
type PlanEntry struct {
	RecordID   string
	ArrayID    string
	Technician string
	Status     OpsStatus
	Priority   OpsPriority
}

// SchedulePlan is the output of the planning pass: ordered jobs plus a
// per-technician workload summary.
type SchedulePlan struct {
	Entries     []PlanEntry
	GeneratedAt string
	ByTech      map[string]int
}

// buildSchedulePlan turns open operations records into maintenance jobs.
// Records already closed are excluded; missing site labels are filled from
// settings.LabelDefaults so every job can be routed to a physical array.
func buildSchedulePlan(records []OpsRecord, settings Settings) (*SchedulePlan, error) {
	plan := &SchedulePlan{
		Entries:     []PlanEntry{},
		GeneratedAt: timeNowOps(),
		ByTech:      map[string]int{},
	}
	for _, record := range records {
		if record.Status == OpsStatusClosed {
			continue
		}
		entry := PlanEntry{
			RecordID:   record.ID,
			ArrayID:    record.LabelValue("site"),
			Technician: record.Owner,
			Status:     record.Status,
			Priority:   record.Priority,
		}
		if entry.ArrayID == "" {
			entry.ArrayID = settings.LabelDefaults["site"]
		}
		if entry.ArrayID == "" {
			return nil, fmt.Errorf("%w: no site assigned for %s", ErrOpsPolicy, record.ID)
		}
		plan.Entries = append(plan.Entries, entry)
		plan.ByTech[record.Owner]++
	}
	return plan, nil
}

// resolveScheduleRule resolves the default site rule through the catalog.
func resolveScheduleRule(settings Settings) (string, error) {
	resolver := newRuleResolver(opsRules())
	if resolver == nil {
		return "", ErrOpsPolicy
	}
	rule, err := resolver.Resolve("OPS-0101")
	if err != nil {
		return "", err
	}
	return rule.Code, nil
}

// planPriorityFilter keeps only jobs at or above the given priority weight.
func planPriorityFilter(entries []PlanEntry, minWeight int) []PlanEntry {
	out := make([]PlanEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.Priority.Weight() >= minWeight {
			out = append(out, entry)
		}
	}
	return out
}

// planByStatus groups jobs by their current status.
func planByStatus(entries []PlanEntry, status OpsStatus) []PlanEntry {
	out := make([]PlanEntry, 0)
	for _, entry := range entries {
		if entry.Status == status {
			out = append(out, entry)
		}
	}
	return out
}

// planSummaryText renders a short human-readable summary of a plan.
func planSummaryText(plan *SchedulePlan) string {
	parts := make([]string, 0, len(plan.Entries))
	for _, entry := range plan.Entries {
		parts = append(parts, strings.Join([]string{entry.RecordID, string(entry.Status)}, ":"))
	}
	return strings.Join(parts, ",")
}
