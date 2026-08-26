package main

import (
	"testing"
)

func opsRecordsFixture() []OpsRecord {
	return []OpsRecord{
		{ID: "op-001", Owner: "lin", Subject: "inverter a03", Status: OpsStatusActive, Priority: OpsPriorityHigh, Labels: map[string]string{"site": "A-03", "evidence": "ok"}},
		{ID: "op-002", Owner: "wang", Subject: "panel clean", Status: OpsStatusQueued, Priority: OpsPriorityNormal, Labels: map[string]string{}},
	}
}

func assertNoPanic(t *testing.T, what string, fn func()) {
	t.Helper()
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("%s panicked: %v", what, recovered)
		}
	}()
	fn()
}

func TestScheduleDefaultSiteApplied(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("buildSchedulePlan panicked: %v", recovered)
		}
	}()
	plan, err := buildSchedulePlan(opsRecordsFixture(), loadSettings())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range plan.Entries {
		if entry.RecordID == "op-002" && entry.ArrayID == "A-01" {
			found = true
		}
	}
	if !found {
		t.Fatalf("default site not applied to record without site: %+v", plan.Entries)
	}
}

func TestScheduleDefensiveNilMap(t *testing.T) {
	assertNoPanic(t, "buildSchedulePlan(nil defaults)", func() {
		_, _ = buildSchedulePlan(opsRecordsFixture(), Settings{LabelDefaults: nil})
	})
}

func TestRuleResolverEmptyCatalog(t *testing.T) {
	assertNoPanic(t, "newRuleResolver(nil).Resolve", func() {
		resolver := newRuleResolver(nil)
		if resolver == nil {
			t.Fatal("resolver should not be nil")
		}
		rule, err := resolver.Resolve("OPS-0101")
		if err == nil {
			t.Fatal("expected not found for empty catalog")
		}
		if rule != nil {
			t.Fatalf("expected nil rule, got %+v", rule)
		}
	})
}

func TestScheduleRuleResolution(t *testing.T) {
	code, err := resolveScheduleRule(loadSettings())
	if err != nil {
		t.Fatal(err)
	}
	if code != "OPS-0101" {
		t.Fatalf("resolved code = %q", code)
	}
}
