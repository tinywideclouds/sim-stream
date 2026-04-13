package simulate_test

import (
	"testing"
	"time"

	"github.com/tinywideclouds/go-power-simulator/internal/aiengine"
	"github.com/tinywideclouds/go-power-simulator/internal/engine"
	"github.com/tinywideclouds/go-power-simulator/internal/simulate"
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

func (m *MockPowerReporter) AddPowerUsage(householdID string, simTime time.Time, totalWatts, indoorTempC, tankTempC float64, activeDevices []string) error {
	m.Calls++
	return nil
}

type MockOrchestrator struct {
	Results []aiengine.TickResult
	TickIdx int
}

func (m *MockOrchestrator) Tick(state *engine.SimulationState, tickDuration time.Duration, weather engine.WeatherProvider, grid engine.GridProvider) aiengine.TickResult {
	if m.TickIdx >= len(m.Results) {
		return aiengine.TickResult{}
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
		Results: []aiengine.TickResult{
			{
				Timestamp: time.Now(),
				ActiveActors: []engine.ActorTickState{
					{ActorID: "actor_1", ActionID: "wfh_session", IsShared: false, Meters: map[string]float64{"energy": 80.0}},
				},
			},
			{
				Timestamp: time.Now(),
				ActiveActors: []engine.ActorTickState{
					{ActorID: "actor_1", ActionID: "wfh_session", IsShared: false, Meters: map[string]float64{"energy": 75.0}},
				},
			},
			{
				Timestamp: time.Now(),
				ActiveActors: []engine.ActorTickState{
					{ActorID: "actor_1", ActionID: "cook_lunch", IsShared: true, Meters: map[string]float64{"energy": 70.0}},
				},
			},
			{
				Timestamp:    time.Now(),
				ActiveActors: []engine.ActorTickState{},
			},
		},
	}

	runner := simulate.NewRunner(mockOrch, mockActorRep, mockPowerRep, mockMeterRep)

	state := &engine.SimulationState{
		Blueprint: &domain.NodeArchetype{ArchetypeID: "test_house"},
	}

	err := runner.Run(state, 4*time.Minute, 1*time.Minute, nil, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if mockPowerRep.Calls != 4 {
		t.Errorf("Expected 4 power logs, got %d", mockPowerRep.Calls)
	}

	if mockMeterRep.Calls != 3 {
		t.Errorf("Expected 3 meter logs, got %d", mockMeterRep.Calls)
	}

	expectedTransitions := []string{
		"actor_1:wfh_session:false",
		"actor_1:cook_lunch:true",
		"actor_1:Idle/Away:false",
	}

	if len(mockActorRep.Calls) != len(expectedTransitions) {
		t.Fatalf("Expected %d actor transitions, got %d", len(expectedTransitions), len(mockActorRep.Calls))
	}

	for i, expected := range expectedTransitions {
		if mockActorRep.Calls[i] != expected {
			t.Errorf("Step %d: Expected %s, got %s", i, expected, mockActorRep.Calls[i])
		}
	}
}
