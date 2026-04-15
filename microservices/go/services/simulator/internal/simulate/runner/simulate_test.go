package runner_test

import (
	"testing"
	"time"

	"github.com/tinywideclouds/go-power-simulator/internal/simulate/core"
	"github.com/tinywideclouds/go-power-simulator/internal/simulate/runner"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

type MockActorReporter struct {
	Calls []string
}

func (m *MockActorReporter) AddActorAction(householdID, actorID, actionID string, isShared bool, simTime time.Time) error {
	sharedStr := "false"
	if isShared {
		sharedStr = "true"
	}
	m.Calls = append(m.Calls, actorID+":"+actionID+":"+sharedStr)
	return nil
}

type MockMeterReporter struct {
	Calls int
}

func (m *MockMeterReporter) AddActorMeters(householdID, actorID string, simTime time.Time, energy, hunger, hygiene, leisure float64) error {
	m.Calls++
	return nil
}

type MockPowerReporter struct {
	Calls int
}

func (m *MockPowerReporter) AddPowerUsage(householdID string, simTime time.Time, totalWatts, indoorTempC, externalTempC, tankTempC float64, activeDevices []string) error {
	m.Calls++
	return nil
}

type MockOrchestrator struct {
	Results []core.TickResult
	TickIdx int
}

func (m *MockOrchestrator) Tick(state *core.SimulationState, tickDuration time.Duration, weather core.WeatherProvider, grid core.GridProvider) core.TickResult {
	if m.TickIdx >= len(m.Results) {
		return core.TickResult{Timestamp: state.SimTime} // Return the current state time
	}
	res := m.Results[m.TickIdx]
	m.TickIdx++
	return res
}

func TestRunner_TimeBasedExecutionAndTransitions(t *testing.T) {
	mockActorRep := &MockActorReporter{}
	mockPowerRep := &MockPowerReporter{}
	mockMeterRep := &MockMeterReporter{}

	mockOrch := &MockOrchestrator{
		Results: []core.TickResult{
			{
				Timestamp: time.Now(),
				ActiveActors: []core.ActorTickState{
					{ActorID: "actor_1", ActionID: "wfh_session", IsShared: false},
				},
				AllHumanMeters: map[string]map[string]float64{"actor_1": {"energy": 80.0}},
			},
			{
				Timestamp: time.Now(),
				ActiveActors: []core.ActorTickState{
					{ActorID: "actor_1", ActionID: "wfh_session", IsShared: false},
				},
				AllHumanMeters: map[string]map[string]float64{"actor_1": {"energy": 75.0}},
			},
			{
				Timestamp: time.Now(),
				ActiveActors: []core.ActorTickState{
					{ActorID: "actor_1", ActionID: "cook_lunch", IsShared: true},
				},
				AllHumanMeters: map[string]map[string]float64{"actor_1": {"energy": 70.0}},
			},
			{
				Timestamp:      time.Now(),
				ActiveActors:   []core.ActorTickState{},
				AllHumanMeters: map[string]map[string]float64{"actor_1": {"energy": 65.0}},
			},
		},
	}

	runner := runner.NewRunner(mockOrch, mockActorRep, mockPowerRep, mockMeterRep)

	state := &core.SimulationState{
		Blueprint: &domain.NodeArchetype{ArchetypeID: "test_house"},
		SimTime:   time.Now(),
	}

	err := runner.Run(state, 4*time.Minute, 1*time.Minute, 1*time.Minute, nil, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if mockPowerRep.Calls != 4 {
		t.Errorf("Expected 4 power logs, got %d", mockPowerRep.Calls)
	}

	if mockMeterRep.Calls != 4 {
		t.Errorf("Expected 4 meter logs, got %d", mockMeterRep.Calls)
	}

	expectedTransitions := []string{
		"actor_1:wfh_session:false",
		"actor_1:cook_lunch:true",
		"actor_1:Idle/Away:false",
	}

	if len(mockActorRep.Calls) != len(expectedTransitions) {
		t.Fatalf("Expected %d actor transitions, got %d", len(expectedTransitions), len(mockActorRep.Calls))
	}
}

// BUGFIX TEST: Ensure the runner actually advances the simulation time.
func TestRunner_AdvancesSimulationTime(t *testing.T) {
	mockOrch := &MockOrchestrator{Results: make([]core.TickResult, 100)}
	run := runner.NewRunner(mockOrch, nil, nil, nil)

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	state := &core.SimulationState{
		Blueprint: &domain.NodeArchetype{ArchetypeID: "time_test"},
		SimTime:   startTime,
	}

	simLength := 10 * time.Minute
	tickInterval := 1 * time.Minute

	err := run.Run(state, simLength, tickInterval, tickInterval, nil, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedEndTime := startTime.Add(simLength)
	if !state.SimTime.Equal(expectedEndTime) {
		t.Errorf("Simulation time did not advance correctly. Expected %v, got %v", expectedEndTime, state.SimTime)
	}
}
