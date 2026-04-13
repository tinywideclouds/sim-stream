package aiengine

import (
	"testing"
	"time"

	"github.com/tinywideclouds/go-power-simulator/internal/engine"
	"github.com/tinywideclouds/go-sim-probability/pkg/generator"
	"github.com/tinywideclouds/go-sim-probability/pkg/parsers"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

type MockAIEngine struct {
	ActiveActors      []engine.ActorTickState
	InterruptedActors []string
	UrgencyScore      float64
}

func (m *MockAIEngine) Process(state *engine.SimulationState, snapshot parsers.StateSnapshot, tickDuration time.Duration) ([]engine.ActorTickState, []string, []string) {
	return m.ActiveActors, []string{}, []string{}
}

func (m *MockAIEngine) InterruptCurrentTask(actorID string, state *engine.SimulationState) bool {
	m.InterruptedActors = append(m.InterruptedActors, actorID)
	return true
}

func (m *MockAIEngine) GetActionUrgency(actorID string, actionID string, state *engine.SimulationState) float64 {
	return m.UrgencyScore
}

func (m *MockAIEngine) ForceTask(actorID string, taskName string, duration time.Duration, startTime time.Time, satisfies map[string]domain.ActionFill) {
}

func (m *MockAIEngine) GetActorSnapshot(actorID string) parsers.StateSnapshot {
	return parsers.StateSnapshot{"actor.energy": 100.0}
}

type MockWeather struct{}

func (m MockWeather) GetTemperature(t time.Time) float64     { return 15.0 }
func (m MockWeather) GetPrecipitation(t time.Time) float64   { return 0.0 }
func (m MockWeather) GetSolarIrradiance(t time.Time) float64 { return 0.0 }

type MockGrid struct{}

func (m MockGrid) NominalVoltage() float64               { return 230.0 }
func (m MockGrid) LiveVoltage(t time.Time) float64       { return 230.0 }
func (m MockGrid) RecordDraw(watts float64, t time.Time) {}

func TestOrchestrator_GatheringWindow(t *testing.T) {
	sampler := generator.NewSampler([32]byte{})
	ai := &MockAIEngine{
		ActiveActors: []engine.ActorTickState{{ActorID: "actor_1", ActionID: "make_tea"}},
		UrgencyScore: 0.5,
	}
	orch := NewOrchestrator(ai, sampler)

	state := &engine.SimulationState{
		SimTime: time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC),
		Actors: map[string]*engine.ActorLedger{
			"actor_1": {},
			"actor_2": {},
		},
		House: engine.HouseholdLedger{
			PendingEvents: make(map[string]*engine.PendingEvent),
			ResourceLocks: make(map[string]string),
		},
		Blueprint: &domain.NodeArchetype{
			Actors: []domain.Actor{
				{ActorID: "actor_1", Type: "adult", AIModel: "utility"},
				{ActorID: "actor_2", Type: "adult", AIModel: "utility"},
			},
			Actions: []domain.ActionTemplate{
				{
					ActionID: "make_tea",
					Sharing: &domain.SharingProfile{
						Type:            domain.SharingScalable,
						GatheringWindow: "5m",
						MaxParticipants: 4,
					},
					Duration: domain.ProbabilityDistribution{Type: domain.DistributionTypeConstant, Value: "3m"},
				},
			},
		},
	}

	orch.Tick(state, time.Minute, MockWeather{}, MockGrid{})

	if len(state.House.PendingEvents) != 1 {
		t.Fatalf("Expected 1 pending event, got %d", len(state.House.PendingEvents))
	}

	var ev *engine.PendingEvent
	for _, v := range state.House.PendingEvents {
		ev = v
	}

	if ev.IsExecuting {
		t.Fatal("Event should be gathering, not executing")
	}

	if len(ev.Participants) != 2 {
		t.Fatalf("Expected actor_2 to automatically join, got %d participants", len(ev.Participants))
	}

	if state.Actors["actor_2"].CurrentCommitment == nil {
		t.Fatal("actor_2 should have a locked CurrentCommitment while waiting")
	}

	orch.Tick(state, 5*time.Minute, MockWeather{}, MockGrid{})

	if !ev.IsExecuting {
		t.Fatal("Event should be executing after gathering window closed")
	}

	if len(ai.InterruptedActors) != 2 {
		t.Fatalf("Both actors should have received System Interrupts to drop their wait tasks, got %d", len(ai.InterruptedActors))
	}
}
