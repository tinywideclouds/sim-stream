// internal/engine/stage_test.go
package engine_test

import (
	"testing"
	"time"

	"github.com/tinywideclouds/go-power-simulator/internal/engine"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

func TestNewSimulationState(t *testing.T) {
	// Setup a mock integrated blueprint
	blueprint := &domain.NodeArchetype{
		ArchetypeID: "test_node",
		BaseTempC:   18.5,
		Devices: []domain.DeviceTemplate{
			{DeviceID: "kettle_1"},
			{DeviceID: "shower_1"},
		},
		Actors: []domain.Actor{
			{ActorID: "parent_1"},
			{ActorID: "teenager_1"},
		},
	}

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	state := engine.NewSimulationState(blueprint, startTime)

	// 1. Check basic initializations
	if state.SimTime != startTime {
		t.Errorf("Expected SimTime %v, got %v", startTime, state.SimTime)
	}
	if state.IndoorTempC != 18.5 {
		t.Errorf("Expected IndoorTempC 18.5, got %v", state.IndoorTempC)
	}
	if state.HotWaterTankC != 55.0 {
		t.Errorf("Expected HotWaterTankC 55.0, got %v", state.HotWaterTankC)
	}

	// 2. Check Device Initialization
	if len(state.Devices) != 2 {
		t.Fatalf("Expected 2 devices in ledger, got %d", len(state.Devices))
	}
	kettle, exists := state.Devices["kettle_1"]
	if !exists {
		t.Fatal("Expected kettle_1 to exist in device ledger")
	}
	// BUGFIX: Devices initialize to Standby, not Off
	if kettle.State != domain.DeviceStateStandby {
		t.Errorf("Expected kettle_1 state to be Standby, got %v", kettle.State)
	}

	// 3. Check Actor Initialization
	if len(state.Actors) != 2 {
		t.Fatalf("Expected 2 actors in ledger, got %d", len(state.Actors))
	}
	parent, exists := state.Actors["parent_1"]
	if !exists {
		t.Fatal("Expected parent_1 to exist in actor ledger")
	}
	if parent.CurrentState != domain.ActorStateAsleep {
		t.Errorf("Expected parent_1 to start Asleep, got %v", parent.CurrentState)
	}
	if parent.CurrentRoutineID != "" {
		t.Errorf("Expected parent_1 to have no active routine, got %s", parent.CurrentRoutineID)
	}
	if parent.Satiety == nil {
		t.Fatal("Expected actor Satiety map to be initialized")
	}
}
