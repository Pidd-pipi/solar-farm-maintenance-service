package main

import (
	"testing"
)

func containsLabel(labels []string, want string) bool {
	for _, label := range labels {
		if label == want {
			return true
		}
	}
	return false
}

func activeRecordWithEvidence() OpsRecord {
	return OpsRecord{
		ID:     "op-001",
		Status: OpsStatusActive,
		Labels: map[string]string{"site": "A-03", "operator": "lin", "evidence": "ok", "reviewed": "yes"},
	}
}

func TestComplianceTerminalEnforced(t *testing.T) {
	report, err := checkRecordCompliance(activeRecordWithEvidence(), opsRules())
	if err != nil {
		t.Fatal(err)
	}
	if report.Failed == 0 {
		t.Fatalf("active record should fail terminal rules; passed=%d failed=%d", report.Passed, report.Failed)
	}
	result, ok := complianceByCode(report, "OPS-0308")
	if !ok {
		t.Fatal("OPS-0308 missing from report")
	}
	if result.Passed {
		t.Fatalf("OPS-0308 terminal rule passed for active record: %+v", result)
	}
}

func TestComplianceReportNotAllPass(t *testing.T) {
	report, err := checkRecordCompliance(activeRecordWithEvidence(), opsRules())
	if err != nil {
		t.Fatal(err)
	}
	if rate := compliancePassRate(report); rate >= 100 {
		t.Fatalf("non-closed record reported as fully compliant: pass rate %d%%", rate)
	}
}

func TestCatalogTerminalRuleData(t *testing.T) {
	rules := opsRules()
	var rule0308 *OpsRule
	for i := range rules {
		if rules[i].Code == "OPS-0308" {
			rule0308 = &rules[i]
			break
		}
	}
	if rule0308 == nil {
		t.Fatal("OPS-0308 missing from catalog")
	}
	if !rule0308.Terminal {
		t.Fatalf("OPS-0308 should be terminal, got Terminal=%v", rule0308.Terminal)
	}
	if !containsLabel(rule0308.RequiredLabels, "evidence") {
		t.Fatalf("OPS-0308 should require evidence, got %v", rule0308.RequiredLabels)
	}
}

func TestCompliancePassRateComputed(t *testing.T) {
	report := &ComplianceReport{Passed: 80, Failed: 20}
	if rate := compliancePassRate(report); rate != 80 {
		t.Fatalf("pass rate = %d, want 80", rate)
	}
}
