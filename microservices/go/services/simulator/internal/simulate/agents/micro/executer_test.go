package micro_test

import (
	"testing"
	"time"

	"github.com/tinywideclouds/go-maths/pkg/probability"
	"github.com/tinywideclouds/go-power-simulator/internal/simulate/agents/micro"
	"github.com/tinywideclouds/go-power-simulator/internal/simulate/core"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

func TestExecutor_AdvanceRoutine_Abort(t *testing.T) {
	baseSampler := probability.NewSampler([32]byte{1})
	distSampler := probability.NewDistributionSampler(baseSampler)
	executor := micro.NewExecutor(distSampler)

	blueprint := &domain.NodeArchetype{
		RoutineTemplates: []domain.RoutineTemplate{
			{
				RoutineID: "quick_prep",
				Tasks:     []string{"morning_shower", "make_toast"},
			},
		},
		Scenarios: []domain.ScenarioTemplate{
			{
				ScenarioID: "morning_shower",
				Actions: []domain.ScenarioAction{
					{
						DeviceID: "shower_1",
						State:    domain.DeviceStateOn,
						Parameters: map[string]probability.SampleSpace{
							"duration": {Type: probability.ConstantDistribution, Const: float64(15 * time.Minute)},
						},
					},
				},
			},
		},
		Actors: []domain.Actor{{ActorID: "teenager_1"}},
		Devices: []domain.DeviceTemplate{
			{DeviceID: "shower_1"},
			{DeviceID: "toaster_1"},
		},
	}

	baseTime := time.Date(2026, 1, 1, 7, 0, 0, 0, time.UTC)
	state := core.NewSimulationState(blueprint, baseTime)

	actor := state.Actors["teenager_1"]
	actor.CurrentState = domain.ActorStateRoutineActive
	actor.CurrentRoutineID = "quick_prep"
	actor.RoutineStepIndex = 0
	actor.StateEndsAt = baseTime

	deadline := baseTime.Add(10 * time.Minute)

	err := executor.AdvanceRoutine(state, "teenager_1", &blueprint.RoutineTemplates[0], deadline)
	if err != nil {
		t.Fatalf("AdvanceRoutine failed: %v", err)
	}

	if state.Devices["shower_1"].State != domain.DeviceStateOn {
		t.Error("Expected shower to be ON")
	}

	// Advance time to force abort
	state.SimTime = deadline
	_ = executor.AdvanceRoutine(state, "teenager_1", &blueprint.RoutineTemplates[0], deadline)

	if actor.CurrentState != domain.ActorStateHomeFree {
		t.Errorf("Expected actor to be aborted, got %v", actor.CurrentState)
	}
}
