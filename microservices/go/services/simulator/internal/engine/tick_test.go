package engine_test

import (
	"testing"
	"time"

	"github.com/tinywideclouds/go-power-simulator/internal/engine"
	"github.com/tinywideclouds/go-sim-probability/pkg/generator"
	"github.com/tinywideclouds/go-sim-probability/pkg/parsers"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

type DummyWeather struct{}

func (d DummyWeather) GetTemperature(t time.Time) float64     { return 5.0 }
func (d DummyWeather) GetPrecipitation(t time.Time) float64   { return 0.0 }
func (d DummyWeather) GetSolarIrradiance(t time.Time) float64 { return 0.0 }

// DummyGrid guarantees 230V for deterministic testing
type DummyGrid struct{}

func (d DummyGrid) NominalVoltage() float64         { return 230.0 }
func (d DummyGrid) LiveVoltage(t time.Time) float64 { return 230.0 }

func TestOrchestrator_Tick(t *testing.T) {
	var seed [32]byte
	sampler := generator.NewSampler(seed)
	sched := engine.NewScheduler(sampler)
	neg := engine.NewNegotiator()
	exec := engine.NewExecutor(sampler)
	orchestrator := engine.NewOrchestrator(sched, neg, exec, 4)

	blueprint := &domain.NodeArchetype{
		RoutineTemplates: []domain.RoutineTemplate{
			{RoutineID: "morning_prep", Tasks: []string{"morning_shower"}},
		},
		Scenarios: []domain.ScenarioTemplate{
			{
				ScenarioID: "morning_shower",
				Actions: []domain.ScenarioAction{
					{
						DeviceID: "shower_1",
						State:    domain.DeviceStateOn,
						Parameters: map[string]domain.ProbabilityDistribution{
							"duration": {Type: domain.DistributionTypeConstant, Value: "15m"},
						},
					},
				},
			},
		},
		Actors: []domain.ActorTemplate{
			{
				ActorID: "parent_1",
				Type:    "adult",
				Routines: []domain.ActorRoutine{
					{
						RoutineID: "morning_prep",
						Trigger:   domain.ProbabilityDistribution{Type: domain.DistributionTypeConstant, Value: "7h0m"},
						Deadline:  domain.ProbabilityDistribution{Type: domain.DistributionTypeConstant, Value: "8h0m"},
					},
				},
			},
			{
				ActorID: "house_system",
				Type:    "system",
			},
		},
		Devices: []domain.DeviceTemplate{
			{
				DeviceID:          "shower_1",
				ElectricalProfile: domain.DeviceProfile{MaxWatts: 0.0, StandbyWatts: 0.0},
				WaterProfile:      &domain.WaterProfile{HotLitersPerMinute: 10.0},
			},
		},
	}

	baseTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	state := engine.NewSimulationState(blueprint, baseTime)

	snap := parsers.EnvironmentSnapshot{}
	orchestrator.BuildDailyPlan(state, baseTime, snap)
	state.SimTime = time.Date(2026, 1, 1, 6, 59, 45, 0, time.UTC)

	weather := DummyWeather{}
	grid := DummyGrid{}

	res := orchestrator.Tick(state, 15*time.Second, weather, grid)

	if len(res.ActiveDevices) != 1 || res.ActiveDevices[0] != "shower_1" {
		t.Fatalf("Expected shower_1 to be active, got %v", res.ActiveDevices)
	}

	if len(res.ActiveActors) != 1 || res.ActiveActors[0] != "parent_1:morning_shower" {
		t.Errorf("Expected parent_1 to be in morning_shower, got %v", res.ActiveActors)
	}
}
