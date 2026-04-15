package orchestrator_test

import (
	"testing"
	"time"

	"github.com/tinywideclouds/go-maths/pkg/probability"
	"github.com/tinywideclouds/go-power-simulator/internal/simulate/core"
	"github.com/tinywideclouds/go-power-simulator/internal/simulate/orchestrator"
)

type MockEngine struct{}

func (m *MockEngine) Process(state *core.SimulationState, snapshot core.StateSnapshot, tickDuration time.Duration) ([]core.ActorTickState, []string, []string) {
	return nil, nil, nil
}
func (m *MockEngine) GetActorSnapshot(actorID string) core.StateSnapshot {
	return core.StateSnapshot{"actor.energy": 100.0}
}

type MockWeather struct{}

func (m MockWeather) GetTemperature(t time.Time) float64     { return 15.0 }
func (m MockWeather) GetPrecipitation(t time.Time) float64   { return 0.0 }
func (m MockWeather) GetSolarIrradiance(t time.Time) float64 { return 0.0 }

type MockGrid struct{}

func (m MockGrid) NominalVoltage() float64         { return 230.0 }
func (m MockGrid) LiveVoltage(t time.Time) float64 { return 230.0 }

func TestOrchestrator_ReapEvents(t *testing.T) {
	baseSampler := probability.NewSampler([32]byte{})
	sampler := probability.NewDistributionSampler(baseSampler)

	mockEngine := &MockEngine{}
	orch := orchestrator.NewOrchestrator(mockEngine, mockEngine, mockEngine, sampler)

	simTime := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	state := &core.SimulationState{
		SimTime: simTime,
		Actors: map[string]*core.ActorLedger{
			"actor_1": {CurrentCommitment: &core.Commitment{ActionID: "test_action"}},
		},
		House: core.HouseholdLedger{
			PendingEvents: map[string]*core.PendingEvent{
				"event_1": {
					ActionID:        "test_action",
					IsExecuting:     true,
					GatheringEndsAt: simTime.Add(-1 * time.Minute), // Expired!
					Participants:    []string{"actor_1"},
				},
			},
		},
	}

	// Trigger the Orchestrator
	orch.Tick(state, time.Minute, MockWeather{}, MockGrid{})

	// Verify the Orchestrator successfully cleaned up the expired events
	if len(state.House.PendingEvents) != 0 {
		t.Fatalf("Expected pending event to be reaped, got %d remaining", len(state.House.PendingEvents))
	}

	if state.Actors["actor_1"].CurrentCommitment != nil {
		t.Error("Expected actor_1 commitment to be cleared")
	}
}
