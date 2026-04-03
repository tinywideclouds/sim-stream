package aiengine

import (
	"testing"
	"time"

	"github.com/tinywideclouds/go-power-simulator/internal/engine"
	"github.com/tinywideclouds/go-sim-probability/pkg/generator"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

func TestUtilityEngine_EmergentBehavior(t *testing.T) {
	// 1. Setup a V3 Blueprint
	blueprint := &domain.NodeArchetype{
		Meters: []domain.MeterTemplate{
			{MeterID: "hunger", Max: 100.0, BaseDecayPerHour: 10.0, Curve: "linear"},
			{MeterID: "energy", Max: 100.0, BaseDecayPerHour: 5.0, Curve: "linear"},
		},
		Actions: []domain.ActionTemplate{
			{
				ActionID:  "cook_dinner",
				DeviceID:  "cooker_1",
				Satisfies: map[string]float64{"hunger": 80.0},
				Costs:     map[string]float64{"energy": 20.0},
				Duration:  domain.ProbabilityDistribution{Type: domain.DistributionTypeConstant, Value: "45m"},
			},
			{
				ActionID:  "microwave_snack",
				DeviceID:  "microwave_1",
				Satisfies: map[string]float64{"hunger": 25.0},
				Costs:     map[string]float64{"energy": 2.0},
				Duration:  domain.ProbabilityDistribution{Type: domain.DistributionTypeConstant, Value: "5m"},
			},
		},
		Actors: []domain.ActorTemplate{
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
			{DeviceID: "microwave_1"},
		},
	}

	state := engine.NewSimulationState(blueprint, time.Now())

	var seed [32]byte
	sampler := generator.NewSampler(seed)
	utilityEngine := NewUtilityEngine(sampler)

	// Tick 1: Evaluating actions.
	// Urgency for hunger = 1.0 - (20/100) = 0.8
	// Cook Dinner Score = (0.8 * 80) - 20(cost) = 44.0
	// Microwave Score = (0.8 * 25) - 2(cost) = 18.0
	// WAIT. Because they are tired, do they microwave?
	// Let's see what the engine actually chooses!

	activeActors, _, _ := utilityEngine.Process(state, nil, 15*time.Second)

	if len(activeActors) != 1 {
		t.Fatalf("Expected 1 active actor, got %d", len(activeActors))
	}

	// The score for Cook Dinner is higher (44 vs 18), so they choose to cook despite the energy cost!
	if activeActors[0] != "wfh_worker:cook_dinner" {
		t.Errorf("Expected wfh_worker to choose cook_dinner, got %s", activeActors[0])
	}

	// Check that the device was locked
	if state.Devices["cooker_1"].State != domain.DeviceStateOn {
		t.Errorf("Expected cooker_1 to be ON")
	}

	// Check that physiological rewards were applied instantly (Hunger went from 20 -> 100)
	currentHunger := utilityEngine.meters["wfh_worker"]["hunger"]
	if currentHunger != 100.0 {
		t.Errorf("Expected hunger to be satisfied to 100.0, got %.1f", currentHunger)
	}
}
