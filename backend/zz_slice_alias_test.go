package main

import (
	"context"
	"testing"
)

func resetRulePoolForTest() {
	opsRule02Labels = make([]string, 3, 5)
	opsRule02Labels[0] = "site"
	opsRule02Labels[1] = "operator"
	opsRule02Labels[2] = "evidence"
}

func containsLabel(labels []string, want string) bool {
	for _, label := range labels {
		if label == want {
			return true
		}
	}
	return false
}

func pageIDs(page OpsPage) []string {
	ids := make([]string, 0, len(page.Items))
	for _, item := range page.Items {
		ids = append(ids, item.ID)
	}
	return ids
}

func TestRuleLabelsIsolatedAfterMutate(t *testing.T) {
	resetRulePoolForTest()
	rules := opsRules()
	var rule0201, rule0202 *OpsRule
	for i := range rules {
		if rules[i].Code == "OPS-0201" {
			rule0201 = &rules[i]
		}
		if rules[i].Code == "OPS-0202" {
			rule0202 = &rules[i]
		}
	}
	if rule0201 == nil || rule0202 == nil {
		t.Fatal("rules missing")
	}
	rule0201.RequiredLabels = append(rule0201.RequiredLabels, "urgent")
	if containsLabel(rule0202.RequiredLabels, "urgent") {
		t.Fatalf("OPS-0202 labels polluted by OPS-0201 mutation: %v", rule0202.RequiredLabels)
	}
	if !containsLabel(rule0202.RequiredLabels, "reviewed") {
		t.Fatalf("OPS-0202 lost reviewed label: %v", rule0202.RequiredLabels)
	}
}

func TestRuleCatalogComplete(t *testing.T) {
	rules := opsRules()
	seen := map[string]bool{}
	dups := []string{}
	has0301 := false
	for _, rule := range rules {
		if seen[rule.Code] {
			dups = append(dups, rule.Code)
		}
		seen[rule.Code] = true
		if rule.Code == "OPS-0301" {
			has0301 = true
		}
	}
	if len(dups) > 0 {
		t.Fatalf("rule catalog has duplicate codes: %v", dups)
	}
	if !has0301 {
		t.Fatal("OPS-0301 missing from catalog")
	}
	if len(rules) != 112 {
		t.Fatalf("rule catalog count = %d, want 112", len(rules))
	}
}

func searchService() *OpsService {
	return newOpsService([]OpsRecord{
		{ID: "op-001", Owner: "lin", Subject: "inverter a03 maintenance", Status: OpsStatusActive, Priority: OpsPriorityHigh, Labels: map[string]string{"site": "A-03"}},
		{ID: "op-002", Owner: "wang", Subject: "panel cleaning batch", Status: OpsStatusQueued, Priority: OpsPriorityNormal, Labels: map[string]string{"site": "A-04"}},
		{ID: "op-003", Owner: "zhao", Subject: "combiner box inspection", Status: OpsStatusPaused, Priority: OpsPriorityLow, Labels: map[string]string{"site": "B-01"}},
	})
}

func TestSearchResultsIsolated(t *testing.T) {
	service := searchService()
	first, err := service.Search(context.Background(), OpsQuery{Subject: "inverter"})
	if err != nil {
		t.Fatal(err)
	}
	firstIDs := pageIDs(first)
	_, err = service.Search(context.Background(), OpsQuery{Subject: "panel"})
	if err != nil {
		t.Fatal(err)
	}
	got := pageIDs(first)
	if len(got) != len(firstIDs) {
		t.Fatalf("first page length changed after second search: %v", got)
	}
	for i := range firstIDs {
		if got[i] != firstIDs[i] {
			t.Fatalf("first page overwritten by later search: want %v got %v", firstIDs, got)
		}
	}
}

func TestClonePageReturnsIndependentCopy(t *testing.T) {
	service := searchService()
	page, err := service.Search(context.Background(), OpsQuery{})
	if err != nil {
		t.Fatal(err)
	}
	clone := opsClonePage(page)
	if len(clone.Items) == 0 {
		t.Fatal("no items")
	}
	clone.Items[0].ID = "mutated"
	if page.Items[0].ID == "mutated" {
		t.Fatalf("clone shares backing array with original page")
	}
}
