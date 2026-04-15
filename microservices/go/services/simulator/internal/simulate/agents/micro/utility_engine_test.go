package micro_test

import (
	"testing"
	"time"

	"github.com/tinywideclouds/go-maths/pkg/geom"
	"github.com/tinywideclouds/go-maths/pkg/probability"
	"github.com/tinywideclouds/go-power-simulator/internal/simulate/agents/micro"
	"github.com/tinywideclouds/go-power-simulator/internal/simulate/core"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

func TestUtilityEngine_ExternalStateManipulation(t *testing.T) {
	blueprint := &domain.NodeArchetype{
		Actors: []domain.Actor{
			{
				ActorID: "test_actor",
				AIModel: "stable",
				StartingMeters: map[string]float64{
					"energy": 50.0,
				},
				Biology: map[string]domain.InstantiatedBiology{
					"energy": {DecayPerHour: 0.0},
				},
			},
		},
	}
	state := core.NewSimulationState(blueprint, time.Now())

	sampler := probability.NewDistributionSampler(probability.NewSampler([32]byte{}))
	utilityEngine := micro.NewUtilityEngine(sampler)

	utilityEngine.Process(state, nil, 1*time.Minute)

	modifiers := map[string]domain.MeterEffect{"energy": {Amount: -40.0}}
	limits := map[string]float64{"energy": 100.0}

	utilityEngine.ApplyModifiersToMeters("test_actor", modifiers, limits)

	snap := utilityEngine.GetActorSnapshot("test_actor")
	currentEnergy := snap["actor.energy"].(float64)
	if currentEnergy > 10.1 {
		t.Errorf("Expected energy ~10.0 after modifier, got %.2f", currentEnergy)
	}
}

func TestUtilityEngine_EmergentBehavior(t *testing.T) {
	blueprint := &domain.NodeArchetype{
		Meters: []domain.MeterTemplate{
			{MeterID: "hunger", Max: 100.0, Curve: geom.Linear},
			{MeterID: "energy", Max: 100.0, Curve: geom.Linear},
		},
		Actions: []domain.ActionTemplate{
			{
				ActionID: "cook_dinner",
				DeviceID: "cooker_1",
				Satisfies: map[string]domain.ActionFill{
					"hunger": {Amount: 80.0, Curve: geom.Linear},
				},
				Costs: map[string]probability.SampleSpace{
					"energy": {Type: probability.ConstantDistribution, Const: 20.0},
				},
				Duration: domain.DynamicDistribution{
					Base: probability.SampleSpace{Type: probability.ConstantDistribution, Const: float64(45 * time.Minute)},
				},
			},
		},
		Actors: []domain.Actor{
			{
				ActorID: "wfh_worker",
				Type:    "adult",
				AIModel: "utility",
				StartingMeters: map[string]float64{
					"hunger": 20.0,
					"energy": 10.0,
				},
				Biology: map[string]domain.InstantiatedBiology{
					"hunger": {DecayPerHour: 10.0},
					"energy": {DecayPerHour: 5.0},
				},
			},
		},
		Devices: []domain.DeviceTemplate{
			{DeviceID: "cooker_1"},
		},
	}

	state := core.NewSimulationState(blueprint, time.Now())

	// Force the CurrentPhase to Home so Utility AI evaluates them!
	state.Actors["wfh_worker"].CurrentPhase = domain.PhaseTypeHome

	sampler := probability.NewDistributionSampler(probability.NewSampler([32]byte{}))
	utilityEngine := micro.NewUtilityEngine(sampler)

	activeActors, _, _ := utilityEngine.Process(state, nil, 15*time.Second)

	if len(activeActors) != 1 {
		t.Fatalf("Expected 1 active actor, got %d", len(activeActors))
	}

	if activeActors[0].ActorID != "wfh_worker" || activeActors[0].ActionID != "cook_dinner" {
		t.Errorf("Expected wfh_worker to choose cook_dinner, got %v", activeActors[0])
	}
}
