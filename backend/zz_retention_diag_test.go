package main

import (
	"testing"
)

func TestAuditTrimKeepsNewest(t *testing.T) {
	audit := newOpsAudit()
	for i := 0; i < 5000; i++ {
		audit.Add("r-1", "created", "lin")
	}
	audit.Trim(100)
	if got := audit.Count(); got != 100 {
		t.Fatalf("audit count after Trim(100) = %d", got)
	}
	if got := audit.Capacity(); got > 256 {
		t.Fatalf("audit capacity not reclaimed after Trim: %d", got)
	}
	latest, ok := audit.Latest()
	if !ok || latest.Type != "created" {
		t.Fatalf("latest event missing after trim: %+v", latest)
	}
}

func TestAuditClearReclaimsCapacity(t *testing.T) {
	audit := newOpsAudit()
	for i := 0; i < 5000; i++ {
		audit.Add("r-1", "created", "lin")
	}
	audit.Clear()
	if got := audit.Capacity(); got > 128 {
		t.Fatalf("audit capacity not reclaimed after Clear: %d", got)
	}
}

func TestStateTrimKeepsNewest(t *testing.T) {
	machine := newOpsStateMachine()
	moves := [][2]OpsStatus{
		{OpsStatusQueued, OpsStatusActive},
		{OpsStatusActive, OpsStatusPaused},
		{OpsStatusPaused, OpsStatusActive},
		{OpsStatusActive, OpsStatusPaused},
	}
	for i := 0; i < 5000; i++ {
		pair := moves[i%len(moves)]
		if err := machine.Move(pair[0], pair[1], "cycle"); err != nil {
			t.Fatal(err)
		}
	}
	machine.TrimHistory(100)
	if got := len(machine.History()); got != 100 {
		t.Fatalf("history count after TrimHistory(100) = %d", got)
	}
	if got := machine.Capacity(); got > 256 {
		t.Fatalf("history capacity not reclaimed after TrimHistory: %d", got)
	}
}

func TestStateResetReclaimsCapacity(t *testing.T) {
	machine := newOpsStateMachine()
	moves := [][2]OpsStatus{
		{OpsStatusQueued, OpsStatusActive},
		{OpsStatusActive, OpsStatusPaused},
		{OpsStatusPaused, OpsStatusActive},
		{OpsStatusActive, OpsStatusPaused},
	}
	for i := 0; i < 5000; i++ {
		pair := moves[i%len(moves)]
		if err := machine.Move(pair[0], pair[1], "cycle"); err != nil {
			t.Fatal(err)
		}
	}
	machine.Reset()
	if got := machine.Capacity(); got > 128 {
		t.Fatalf("history capacity not reclaimed after Reset: %d", got)
	}
}
