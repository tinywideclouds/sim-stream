// internal/engine/executer_test.go
package engine_test

import (
	"testing"
	"time"

	"github.com/tinywideclouds/go-power-simulator/internal/engine"
	"github.com/tinywideclouds/go-sim-probability/pkg/generator"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

func TestExecutor_AdvanceRoutine_Abort(t *testing.T) {
	// 1. Setup Deterministic Math
	var seed [32]byte
	seed[0] = 1
	sampler := generator.NewSampler(seed)
	executor := engine.NewExecutor(sampler)

	// 2. Setup Blueprint with a Routine and two Scenarios
	blueprint := &domain.NodeArchetype{
		RoutineTemplates: []domain.RoutineTemplate{
			{
				RoutineID: "quick_prep",
				Tasks:     []string{"morning_shower", "make_toast"}, // 2 tasks
			},
		},
		Scenarios: []domain.ScenarioTemplate{
			{
				ScenarioID: "morning_shower",
				Actions: []domain.ScenarioAction{
					{
						DeviceID: "shower_1",
						State:    domain.DeviceStateOn,
						Parameters: map[string]domain.ProbabilityDistribution{
							"duration": {Type: domain.DistributionTypeConstant, Value: "15m"}, // Takes 15 mins
						},
					},
				},
			},
			{
				ScenarioID: "make_toast",
				Actions: []domain.ScenarioAction{
					{
						DeviceID: "toaster_1",
						State:    domain.DeviceStateOn,
						Parameters: map[string]domain.ProbabilityDistribution{
							"duration": {Type: domain.DistributionTypeConstant, Value: "5m"},
						},
					},
				},
			},
		},
		Actors: []domain.ActorTemplate{{ActorID: "teenager_1"}},
		Devices: []domain.DeviceTemplate{
			{DeviceID: "shower_1"},
			{DeviceID: "toaster_1"},
		},
	}

	// 3. Initialize the State Ledger
	baseTime := time.Date(2026, 1, 1, 7, 0, 0, 0, time.UTC)
	state := engine.NewSimulationState(blueprint, baseTime)

	// Manually trigger the actor into the routine
	actor := state.Actors["teenager_1"]
	actor.CurrentState = domain.ActorStateRoutineActive
	actor.CurrentRoutineID = "quick_prep"
	actor.RoutineStepIndex = 0
	actor.StateEndsAt = baseTime // Ready immediately

	// DEADLINE: They only have 10 minutes left before they have to leave!
	deadline := baseTime.Add(10 * time.Minute)

	// --- TICK 1: (07:00:00) ---
	err := executor.AdvanceRoutine(state, "teenager_1", &blueprint.RoutineTemplates[0], deadline)
	if err != nil {
		t.Fatalf("AdvanceRoutine failed: %v", err)
	}

	// Verify Task 1 (Shower) started
	if state.Devices["shower_1"].State != domain.DeviceStateOn {
		t.Error("Expected shower to be ON")
	}
	if actor.RoutineStepIndex != 1 {
		t.Errorf("Expected step index to advance to 1, got %d", actor.RoutineStepIndex)
	}

	// --- TICK 2: 10 MINUTES LATER (07:10:00) ---
	// Fast forward the clock to the deadline
	state.SimTime = deadline

	err = executor.AdvanceRoutine(state, "teenager_1", &blueprint.RoutineTemplates[0], deadline)
	if err != nil {
		t.Fatalf("AdvanceRoutine failed: %v", err)
	}

	// Verify the Routine was violently aborted
	if actor.CurrentState != domain.ActorStateHomeFree {
		t.Errorf("Expected actor to be aborted/freed, got %v", actor.CurrentState)
	}
	if actor.CurrentRoutineID != "" {
		t.Errorf("Expected RoutineID to be cleared, got %s", actor.CurrentRoutineID)
	}

	// Verify Task 2 (Toast) NEVER turned on!
	if state.Devices["toaster_1"].State == domain.DeviceStateOn {
		t.Error("Expected toaster to remain OFF because the routine was aborted!")
	}
}
