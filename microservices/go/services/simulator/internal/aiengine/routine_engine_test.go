package aiengine

import (
	"testing"
	"time"

	"github.com/tinywideclouds/go-power-simulator/internal/engine"
	"github.com/tinywideclouds/go-sim-probability/pkg/parsers"
)

func TestRoutineEngine_Adapters(t *testing.T) {
	// Initialize with nil dependencies as we only test the adapter logic
	re := NewRoutineEngine(nil, nil, nil, 3)

	actorID := "test_actor"
	simTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	// Inject a manual plan into the private map for testing
	re.dailyPlan[actorID] = make(map[string]engine.ScheduledRoutine)
	re.dailyPlan[actorID]["cook_dinner"] = engine.ScheduledRoutine{
		RoutineID:      "cook_dinner",
		TargetStart:    simTime.Add(-1 * time.Hour), // Started 1 hour ago
		TargetDeadline: simTime.Add(1 * time.Hour),  // Ends in 1 hour
		HasStarted:     false,
	}

	snap := make(parsers.StateSnapshot)

	// 1. GetActiveRoutineAction
	actionID, routineID, isScheduled := re.GetActiveRoutineAction(actorID, simTime, snap)
	if !isScheduled || actionID != "cook_dinner" || routineID != "cook_dinner" {
		t.Errorf("Expected scheduled cook_dinner, got actionID: %s, isScheduled: %v", actionID, isScheduled)
	}

	// 2. AbortRoutine (Should clear the TargetDeadline)
	re.AbortRoutine(actorID)

	// 3. Verify it is no longer scheduled
	_, _, isScheduledNow := re.GetActiveRoutineAction(actorID, simTime, snap)
	if isScheduledNow {
		t.Errorf("Expected routine to be aborted and no longer scheduled")
	}
}
