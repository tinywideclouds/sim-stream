package world_test

import (
	"testing"
	"time"

	"github.com/tinywideclouds/go-power-simulator/internal/engine"
	"github.com/tinywideclouds/go-power-simulator/internal/world"
	"github.com/tinywideclouds/go-sim-probability/pkg/generator"
	"github.com/tinywideclouds/go-sim-probability/pkg/parsers"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

// -- MOCKS --
type MockCalendar struct{}

func (m *MockCalendar) GetDayType(date time.Time) string { return "workday" }
func (m *MockCalendar) IsHoliday(date time.Time) bool    { return false }

type MockUtility struct {
	canAfford bool
}

func (m *MockUtility) ResetMeters(actorID string, starting map[string]float64) {}

func (m *MockUtility) ApplyModifiersToMeters(actorID string, mods map[string]domain.ContinuousEffect, limits map[string]float64) {
}
func (m *MockUtility) Process(state *engine.SimulationState, snapshot parsers.StateSnapshot, tick time.Duration) ([]engine.ActorTickState, []string, []string) {
	return nil, nil, nil
}
func (m *MockUtility) HasMeters(actorID string, costs map[string]float64) bool { return m.canAfford }
func (m *MockUtility) GetInterruptAction(actorID string, state *engine.SimulationState) string {
	return ""
}
func (m *MockUtility) GetActionUrgency(actorID string, actionID string, state *engine.SimulationState) float64 {
	return 0.5
}
func (m *MockUtility) GetActorSnapshot(actorID string) parsers.StateSnapshot {
	return parsers.StateSnapshot{"actor.energy": 50.0}
}

func (m *MockUtility) InterruptCurrentTask(actorID string, state *engine.SimulationState) bool {
	return true
}

func (m *MockUtility) ForceTask(actorID string, actionID string, duration time.Duration, startTime time.Time, satisfies map[string]domain.ActionFill) {
}

type MockRoutine struct {
	hasRoutine      bool
	intendedAction  string
	intendedRoutine string
}

func (m *MockRoutine) GetActiveRoutineAction(actorID string, simTime time.Time, snap parsers.StateSnapshot) (string, string, bool) {
	return m.intendedAction, m.intendedRoutine, m.hasRoutine
}
func (m *MockRoutine) ProcessActor(actorID string, state *engine.SimulationState, tick time.Duration, snap parsers.StateSnapshot) string {
	return "cooking"
}
func (m *MockRoutine) AbortRoutine(actorID string) {}

// -----------

func TestStableEngine_Arbitration_Success(t *testing.T) {
	blueprint := &domain.NodeArchetype{
		Actors: []domain.Actor{{ActorID: "parent", AIModel: "stable"}},
		Actions: []domain.ActionTemplate{
			{ActionID: "cook_dinner", Costs: map[string]float64{"energy": 25.0}},
		},
		CollectiveEvents: []domain.CollectiveEvent{
			{EventID: "family_dinner"},
		},
	}
	state := engine.NewSimulationState(blueprint, time.Now())
	state.Actors["parent"] = &engine.ActorLedger{CurrentState: domain.ActorStateHomeFree}

	mockUtil := &MockUtility{canAfford: true}
	mockRoutine := &MockRoutine{hasRoutine: true, intendedAction: "cook_dinner", intendedRoutine: "family_dinner"}

	stableEngine := world.NewStableEngine(mockUtil, mockRoutine, &MockCalendar{}, generator.NewSampler([32]byte{}))

	active, _, _ := stableEngine.Process(state, make(parsers.StateSnapshot), 1*time.Minute)

	if len(active) == 0 || active[0].ActorID != "parent" || active[0].ActionID != "cooking" {
		t.Errorf("Expected actor to successfully execute V2 routine, got %v", active)
	}
}

func TestStableEngine_Arbitration_Failure(t *testing.T) {
	blueprint := &domain.NodeArchetype{
		Actors: []domain.Actor{{ActorID: "exhausted_parent", AIModel: "stable"}},
		Actions: []domain.ActionTemplate{
			{ActionID: "cook_dinner", Costs: map[string]float64{"energy": 25.0}},
		},
		CollectiveEvents: []domain.CollectiveEvent{
			{EventID: "family_dinner"},
		},
	}
	state := engine.NewSimulationState(blueprint, time.Now())
	state.Actors["exhausted_parent"] = &engine.ActorLedger{CurrentState: domain.ActorStateHomeFree}

	mockUtil := &MockUtility{canAfford: false}
	mockRoutine := &MockRoutine{hasRoutine: true, intendedAction: "cook_dinner", intendedRoutine: "family_dinner"}

	stableEngine := world.NewStableEngine(mockUtil, mockRoutine, &MockCalendar{}, generator.NewSampler([32]byte{}))

	_, anomalies, debugLogs := stableEngine.Process(state, make(parsers.StateSnapshot), 1*time.Minute)

	if len(anomalies) == 0 || anomalies[0] != "exhausted_parent:aborted_routine:family_dinner" {
		t.Errorf("Expected routine to abort due to V3 constraint failure, got anomalies: %v", anomalies)
	}

	foundLog := false
	for _, l := range debugLogs {
		if l == "[exhausted_parent] V3 Rejected Routine 'family_dinner': Insufficient meters. Burnout at 15.0" {
			foundLog = true
		}
	}
	if !foundLog {
		t.Errorf("Expected V3 rejection log, not found in %v", debugLogs)
	}
}

func TestStableEngine_AwayGhosting(t *testing.T) {
	blueprint := &domain.NodeArchetype{
		Actors: []domain.Actor{
			{
				ActorID: "office_worker",
				AIModel: "stable",
				Phases: []domain.Phase{
					{
						PhaseID:    "work_shift",
						AnchorTime: "08:30",
						Type:       "away",
						Duration: domain.PhaseDuration{
							ProbabilityDistribution: domain.ProbabilityDistribution{Type: domain.DistributionTypeConstant, Value: "8h"},
						},
					},
				},
			},
		},
	}

	startTime := time.Date(2026, 1, 1, 8, 29, 0, 0, time.UTC)
	state := engine.NewSimulationState(blueprint, startTime)
	state.Actors["office_worker"] = &engine.ActorLedger{CurrentState: domain.ActorStateHomeFree}

	sampler := generator.NewSampler([32]byte{})
	stableEngine := world.NewStableEngine(&MockUtility{}, &MockRoutine{}, &MockCalendar{}, sampler)
	snapshot := make(parsers.StateSnapshot)
	tickDuration := 1 * time.Minute

	state.SimTime = state.SimTime.Add(tickDuration)
	stableEngine.Process(state, snapshot, tickDuration)

	ledger := state.Actors["office_worker"]
	if ledger.CurrentState != domain.ActorStateAway {
		t.Errorf("Expected actor to be Away, got %v", ledger.CurrentState)
	}
}

func TestStableEngine_SleepGhosting(t *testing.T) {
	blueprint := &domain.NodeArchetype{
		Actors: []domain.Actor{
			{
				ActorID: "tired_child",
				AIModel: "stable",
				Phases: []domain.Phase{
					{
						PhaseID:    "night_sleep",
						AnchorTime: "20:00",
						Type:       "sleep",
						Duration: domain.PhaseDuration{
							ProbabilityDistribution: domain.ProbabilityDistribution{Type: domain.DistributionTypeConstant, Value: "10h"},
							Flexibility:             2 * time.Hour,
						},
					},
				},
			},
		},
	}

	startTime := time.Date(2026, 1, 1, 19, 59, 0, 0, time.UTC)
	state := engine.NewSimulationState(blueprint, startTime)
	state.Actors["tired_child"] = &engine.ActorLedger{CurrentState: domain.ActorStateHomeFree}

	sampler := generator.NewSampler([32]byte{})
	stableEngine := world.NewStableEngine(&MockUtility{}, &MockRoutine{}, &MockCalendar{}, sampler)
	snapshot := make(parsers.StateSnapshot)
	tickDuration := 1 * time.Minute

	state.SimTime = state.SimTime.Add(tickDuration)
	stableEngine.Process(state, snapshot, tickDuration)

	ledger := state.Actors["tired_child"]
	if ledger.CurrentState != domain.ActorStateAsleep {
		t.Errorf("Expected actor to be Asleep, got %v", ledger.CurrentState)
	}
}

func TestCalculatePhaseTimes_Sleep(t *testing.T) {
	se := world.NewStableEngine(&MockUtility{}, &MockRoutine{}, &MockCalendar{}, generator.NewSampler([32]byte{}))

	state := &engine.SimulationState{
		SimTime: time.Date(2026, 1, 1, 22, 0, 0, 0, time.UTC),
		Blueprint: &domain.NodeArchetype{
			Meters: []domain.MeterTemplate{{MeterID: "energy", Max: 100.0}},
		},
	}

	actor := domain.Actor{
		ActorID: "test_actor",
		Phases: []domain.Phase{
			{Type: "away"},
		},
	}

	phase := domain.Phase{
		Type:       "sleep",
		AnchorTime: "23:00",
		Duration: domain.PhaseDuration{
			ProbabilityDistribution: domain.ProbabilityDistribution{
				Type:  domain.DistributionTypeConstant,
				Value: "8h",
			},
			Flexibility: 2 * time.Hour,
		},
	}

	start, end, _, _ := se.CalculatePhaseTimes(actor, phase, state, "workday")

	expectedStart := time.Date(2026, 1, 1, 23, 0, 0, 0, time.UTC)
	expectedEnd := expectedStart.Add(8 * time.Hour)

	if !start.Equal(expectedStart) {
		t.Errorf("Expected start %v, got %v", expectedStart, start)
	}
	if !end.Equal(expectedEnd) {
		t.Errorf("Expected end %v, got %v", expectedEnd, end)
	}
}
