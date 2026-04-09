// aiengine/utility_engine_test.go
package aiengine

import (
	"testing"
	"time"

	"github.com/tinywideclouds/go-power-simulator/internal/engine"
	"github.com/tinywideclouds/go-sim-probability/pkg/generator"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

func TestUtilityEngine_ExternalStateManipulation(t *testing.T) {
	// Setup minimalist blueprint
	blueprint := &domain.NodeArchetype{
		Actors: []domain.Actor{
			{
				ActorID: "test_actor",
				AIModel: "stable",
				StartingMeters: map[string]float64{
					"energy": 50.0,
				},
			},
		},
	}
	state := engine.NewSimulationState(blueprint, time.Now())

	sampler := generator.NewSampler([32]byte{})
	utilityEngine := NewUtilityEngine(sampler)

	// Trigger initial map population
	utilityEngine.Process(state, nil, 1*time.Minute)

	// 1. Test Modifiers (Re-entry simulation) - UPDATED TO CONTINUOUS EFFECT
	modifiers := map[string]domain.ContinuousEffect{"energy": {Amount: -40.0}}
	limits := map[string]float64{"energy": 100.0}

	utilityEngine.ApplyModifiersToMeters("test_actor", modifiers, limits)

	// Fast check ignoring the tiny decay that happened during Process
	currentEnergy := utilityEngine.meters["test_actor"]["energy"]
	if currentEnergy > 10.1 { // Should be ~10.0 (50 - 40)
		t.Errorf("Expected energy ~10.0 after modifier, got %.2f", currentEnergy)
	}

	// 2. Test Hard Reset (Burnout Snap)
	utilityEngine.ResetMeters("test_actor", map[string]float64{"energy": 100.0})

	if utilityEngine.meters["test_actor"]["energy"] != 100.0 {
		t.Errorf("Expected energy 100.0 after reset, got %.2f", utilityEngine.meters["test_actor"]["energy"])
	}
}

func TestUtilityEngine_EmergentBehavior(t *testing.T) {
	// Setup Blueprint with upgraded ActionFill struct
	blueprint := &domain.NodeArchetype{
		Meters: []domain.MeterTemplate{
			{MeterID: "hunger", Max: 100.0, BaseDecayPerHour: 10.0, Curve: "linear"},
			{MeterID: "energy", Max: 100.0, BaseDecayPerHour: 5.0, Curve: "linear"},
		},
		Actions: []domain.ActionTemplate{
			{
				ActionID: "cook_dinner",
				DeviceID: "cooker_1",
				Satisfies: map[string]domain.ActionFill{
					"hunger": {Amount: 80.0, Curve: "linear"},
				},
				Costs:    map[string]float64{"energy": 20.0},
				Duration: domain.ProbabilityDistribution{Type: domain.DistributionTypeConstant, Value: "45m"},
			},
		},
		Actors: []domain.Actor{
			{
				ActorID: "wfh_worker",
				Type:    "adult",
				AIModel: "utility",
				StartingMeters: map[string]float64{
					"hunger": 20.0, // VERY HUNGRY (Max 100)
					"energy": 10.0, // VERY TIRED
				},
			},
		},
		Devices: []domain.DeviceTemplate{
			{DeviceID: "cooker_1"},
		},
	}

	state := engine.NewSimulationState(blueprint, time.Now())
	sampler := generator.NewSampler([32]byte{})
	utilityEngine := NewUtilityEngine(sampler)

	activeActors, _, _ := utilityEngine.Process(state, nil, 15*time.Second)

	if len(activeActors) != 1 {
		t.Fatalf("Expected 1 active actor, got %d", len(activeActors))
	}

	if activeActors[0] != "wfh_worker:cook_dinner" {
		t.Errorf("Expected wfh_worker to choose cook_dinner, got %s", activeActors[0])
	}
}

func TestUtilityEngine_GetActionUrgency(t *testing.T) {
	blueprint := &domain.NodeArchetype{
		Meters: []domain.MeterTemplate{
			{MeterID: "hunger", Max: 100.0},
		},
		Actions: []domain.ActionTemplate{
			{
				ActionID: "cook_meal",
				Satisfies: map[string]domain.ActionFill{
					"hunger": {Amount: 60.0},
				},
			},
		},
		Actors: []domain.Actor{
			{ActorID: "hungry_actor", AIModel: "stable"},
			{ActorID: "full_actor", AIModel: "stable"},
		},
	}

	state := engine.NewSimulationState(blueprint, time.Now())
	sampler := generator.NewSampler([32]byte{})
	ue := NewUtilityEngine(sampler)

	ue.Process(state, nil, 1*time.Minute)
	ue.meters["hungry_actor"]["hunger"] = 20.0 // Deficit: 80
	ue.meters["full_actor"]["hunger"] = 100.0  // Deficit: 0

	hungryUrgency := ue.GetActionUrgency("hungry_actor", "cook_meal", state)
	expectedHungry := 48.0
	if hungryUrgency != expectedHungry {
		t.Errorf("Expected hungry urgency to be %.1f, got %.1f", expectedHungry, hungryUrgency)
	}

	fullUrgency := ue.GetActionUrgency("full_actor", "cook_meal", state)
	expectedFull := 0.0
	if fullUrgency != expectedFull {
		t.Errorf("Expected full urgency to be %.1f, got %.1f", expectedFull, fullUrgency)
	}
}

func TestUtilityEngine_GetActorSnapshot(t *testing.T) {
	sampler := generator.NewSampler([32]byte{})
	ue := NewUtilityEngine(sampler)

	actorID := "test_actor"
	ue.meters[actorID] = map[string]float64{
		"energy": 80.0,
		"hunger": 30.0,
	}

	// 1. Fetch the localized snapshot
	snapshot := ue.GetActorSnapshot(actorID)
	if snapshot == nil {
		t.Fatalf("Expected a snapshot, got nil")
	}

	// 2. Verify prefix mapping
	if val, ok := snapshot["actor.energy"].(float64); !ok || val != 80.0 {
		t.Errorf("Expected actor.energy to be 80.0, got %v", snapshot["actor.energy"])
	}
	if val, ok := snapshot["actor.hunger"].(float64); !ok || val != 30.0 {
		t.Errorf("Expected actor.hunger to be 30.0, got %v", snapshot["actor.hunger"])
	}

	// 3. Verify internal state protection
	snapshot["actor.energy"] = 10.0 // Attempt to mutate
	if ue.meters[actorID]["energy"] == 10.0 {
		t.Errorf("GetActorSnapshot leaked internal state reference! Internal map was mutated.")
	}
}

// --- UPDATED FOR NEW SIGNATURE ---
func TestUtilityEngine_ForceTask(t *testing.T) {
	sampler := generator.NewSampler([32]byte{})
	ue := NewUtilityEngine(sampler)

	blueprint := &domain.NodeArchetype{
		Actors: []domain.Actor{
			{ActorID: "actor1", StartingMeters: map[string]float64{"energy": 50.0}},
		},
		Meters: []domain.MeterTemplate{
			{MeterID: "energy", Max: 100.0, BaseDecayPerHour: 6.0, Curve: "linear"},
		},
		Actions: []domain.ActionTemplate{
			{
				ActionID: "night_sleep",
				Satisfies: map[string]domain.ActionFill{
					"energy": {Amount: 50.0, Curve: "linear"},
				},
			},
		},
	}

	simTime := time.Date(2026, 1, 1, 23, 0, 0, 0, time.UTC)
	state := engine.NewSimulationState(blueprint, simTime)
	state.Actors["actor1"] = &engine.ActorLedger{CurrentState: domain.ActorStateAsleep}

	ue.ResetMeters("actor1", blueprint.Actors[0].StartingMeters)

	sleepDuration := 8 * time.Hour
	// Note: We supply the map directly, avoiding reliance on looking it up from engine state
	satisfies := map[string]domain.ActionFill{
		"energy": {Amount: 50.0, Curve: "linear"},
	}

	ue.ForceTask("actor1", "night_sleep", sleepDuration, state.SimTime, satisfies)

	tickDuration := 1 * time.Minute
	state.SimTime = state.SimTime.Add(tickDuration)
	activeActors, _, _ := ue.Process(state, nil, tickDuration)

	if len(activeActors) == 0 || activeActors[0] != "actor1:night_sleep" {
		t.Errorf("Expected actor to report night_sleep, got %v", activeActors)
	}

	snap := ue.GetActorSnapshot("actor1")
	energy, exists := snap["actor.energy"]
	if !exists {
		t.Fatalf("Expected actor.energy in snapshot")
	}

	// 50.0 start + (~0.104 fill) - (0.1 decay) = ~50.004
	if energy.(float64) <= 50.0 {
		t.Errorf("Expected energy to start recovering during forced task, got %f", energy.(float64))
	}
}
