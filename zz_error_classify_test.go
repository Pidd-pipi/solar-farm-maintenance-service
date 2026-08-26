package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestErrorChainPreserved002(t *testing.T) {
	wrapped := wrapOps("create", "store.put", ErrOpsConflict)
	if !errors.Is(wrapped, ErrOpsConflict) {
		t.Fatalf("errors.Is(wrapped, ErrOpsConflict) = false: %v", wrapped)
	}
}

func TestOpsCodeSentinelClassifies002(t *testing.T) {
	if got := opsCode(ErrOpsConflict); got != "conflict" {
		t.Fatalf("opsCode(ErrOpsConflict) = %q, want conflict", got)
	}
	if got := opsCode(ErrOpsNotFound); got != "not_found" {
		t.Fatalf("opsCode(ErrOpsNotFound) = %q, want not_found", got)
	}
}

func TestCreateConflictRejected002(t *testing.T) {
	server := httptest.NewServer(newOpsHandler())
	defer server.Close()
	create := func(body string) int {
		response, err := http.Post(server.URL+"/api/ops/records", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		return response.StatusCode
	}
	body := `{"subject":"dup job","owner":"lin","status":"queued","priority":"high","labels":{"site":"A-09"}}`
	if got := create(body); got != http.StatusCreated {
		t.Fatalf("first create: got %d", got)
	}
	if got := create(body); got != http.StatusConflict {
		t.Fatalf("duplicate create should be 409, got %d", got)
	}
}

func TestTransitionRevisionEnforced002(t *testing.T) {
	server := httptest.NewServer(newOpsHandler())
	defer server.Close()
	response, err := http.Post(server.URL+"/api/ops/records/op-001/transition", "application/json", strings.NewReader(`{"target":"paused","expected":999,"actor":"lin"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("revision conflict should be 409, got %d", response.StatusCode)
	}
}
