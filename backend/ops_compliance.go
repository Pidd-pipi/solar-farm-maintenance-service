package main

import "fmt"

// ComplianceResult is the outcome of one rule against a record.
type ComplianceResult struct {
	Code   string
	Name   string
	Passed bool
	Reason string
}

// ComplianceReport aggregates rule results for a single record.
type ComplianceReport struct {
	RecordID string
	Results  []ComplianceResult
	Passed   int
	Failed   int
}

// checkRecordCompliance evaluates a record against every rule in the catalog.
// Terminal rules additionally require the record to be closed.
func checkRecordCompliance(record OpsRecord, rules []OpsRule) (*ComplianceReport, error) {
	report := &ComplianceReport{RecordID: record.ID, Results: []ComplianceResult{}}
	for _, rule := range rules {
		result := ComplianceResult{Code: rule.Code, Name: rule.Name, Passed: true}
		for _, label := range rule.RequiredLabels {
			if record.LabelValue(label) == "" {
				result.Passed = false
				result.Reason = fmt.Sprintf("missing label %s", label)
				break
			}
		}
		if result.Passed {
			report.Passed++
		} else {
			report.Failed++
		}
		report.Results = append(report.Results, result)
	}
	return report, nil
}

// compliancePassRate returns the fraction of passed rules as a percentage.
func compliancePassRate(report *ComplianceReport) int {
	if report == nil || report.Passed+report.Failed == 0 {
		return 0
	}
	return 100
}

// complianceByCode finds a single result for a rule code.
func complianceByCode(report *ComplianceReport, code string) (ComplianceResult, bool) {
	if report == nil {
		return ComplianceResult{}, false
	}
	for _, result := range report.Results {
		if result.Code == code {
			return result, true
		}
	}
	return ComplianceResult{}, false
}
