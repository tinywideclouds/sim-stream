package macro_test

import (
	"testing"
	"time"

	"github.com/tinywideclouds/go-maths/pkg/probability"
	"github.com/tinywideclouds/go-power-simulator/internal/simulate/agents/macro"
	"github.com/tinywideclouds/go-power-simulator/internal/simulate/core"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

type MockCalendar struct{}

func (m *MockCalendar) GetDayType(date time.Time) string { return "workday" }
func (m *MockCalendar) IsHoliday(date time.Time) bool    { return false }

type MockUtility struct {
	canAfford bool
}

func (m *MockUtility) ResetMeters(actorID string, starting map[string]float64) {}
func (m *MockUtility) ApplyModifiersToMeters(actorID string, mods map[string]domain.MeterEffect, limits map[string]float64) {
}
func (m *MockUtility) Process(state *core.SimulationState, snapshot core.StateSnapshot, tick time.Duration) ([]core.ActorTickState, []string, []string) {
	return nil, nil, nil
}
func (m *MockUtility) HasMeters(actorID string, costs map[string]probability.SampleSpace) bool {
	return m.canAfford
}
func (m *MockUtility) GetInterruptAction(actorID string, state *core.SimulationState) string {
	return ""
}
func (m *MockUtility) GetActionUrgency(actorID string, actionID string, state *core.SimulationState) float64 {
	return 0.5
}
func (m *MockUtility) GetActorSnapshot(actorID string) core.StateSnapshot {
	return core.StateSnapshot{"actor.energy": 50.0}
}
func (m *MockUtility) InterruptCurrentTask(actorID string, state *core.SimulationState) bool {
	return true
}
func (m *MockUtility) ForceTask(actorID string, actionID string, duration time.Duration, startTime time.Time, satisfies map[string]domain.ActionFill) {
}

func TestStableEngine_MacroState_Transitions(t *testing.T) {
	blueprint := &domain.NodeArchetype{
		Actors: []domain.Actor{
			{
				ActorID: "office_worker",
				AIModel: "stable",
				Phases: []domain.Phase{
					{
						PhaseID:    "work_shift",
						AnchorTime: "08:30",
						Type:       domain.PhaseTypeAway, // They are leaving the house
						Blocks: []domain.PhaseBlock{
							{
								Probability: 1.0,
								Duration: domain.DynamicDistribution{
									Base: probability.SampleSpace{Type: probability.ConstantDistribution, Const: float64(8 * time.Hour)},
								},
							},
						},
					},
				},
			},
		},
	}

	// Set time to just BEFORE they leave for work
	startTime := time.Date(2026, 1, 1, 8, 29, 0, 0, time.UTC)
	state := core.NewSimulationState(blueprint, startTime)

	// They start at home
	state.Actors["office_worker"] = &core.ActorLedger{
		CurrentState: domain.ActorStateHomeFree,
		CurrentPhase: domain.PhaseTypeHome,
	}

	baseSampler := probability.NewSampler([32]byte{})
	sampler := probability.NewDistributionSampler(baseSampler)

	// Note: We use a nil MockRoutine since stable.go only calls it if AIModel == "routine"
	stableEngine := macro.NewStableEngine(&MockUtility{}, &MockCalendar{}, sampler)
	snapshot := make(core.StateSnapshot)

	// TICK 1: Fast forward 2 minutes. It is now 08:31. The Phase should trigger.
	tickDuration := 2 * time.Minute
	state.SimTime = state.SimTime.Add(tickDuration)

	stableEngine.Process(state, snapshot, tickDuration)

	ledger := state.Actors["office_worker"]

	// 1. Verify Macro-State shifted to Away!
	if ledger.CurrentPhase != domain.PhaseTypeAway {
		t.Errorf("Expected actor Macro-Phase to be Away, got %v", ledger.CurrentPhase)
	}

	// 2. Verify Micro-State shifted to RoutineActive (The "Black Box")
	if ledger.CurrentState != domain.ActorStateRoutineActive {
		t.Errorf("Expected actor Micro-State to be RoutineActive, got %v", ledger.CurrentState)
	}
}
