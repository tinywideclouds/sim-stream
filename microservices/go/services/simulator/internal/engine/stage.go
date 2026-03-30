package engine

import (
	"time"

	"github.com/tinywideclouds/go-sim-schema/domain"
)

// SimulationState represents the exact reality of a house at a given microsecond.
type SimulationState struct {
	Blueprint *domain.NodeArchetype
	SimTime   time.Time

	// Thermodynamics
	IndoorTempC float64

	// Ledgers tracking the physical and human actors
	Devices map[string]*DeviceLedger // Key: device_id
	Actors  map[string]*ActorLedger  // Key: actor_id

	// Traceability: Active UUIDs to attach to the meter readings
	ActiveEventIDs []string
}

// DeviceLedger tracks the physical state and cooldowns of an appliance.
type DeviceLedger struct {
	State         domain.DeviceState
	ActiveEventID string    // The UUID of the scenario that turned this on
	StateEndsAt   time.Time // When does the kettle finish boiling?
	CooldownUntil time.Time // When is the motor allowed to turn on again?
}

// ActorLedger tracks human availability and satiety/fatigue.
type ActorLedger struct {
	IsBusy    bool
	BusyUntil time.Time

	// Fatigue tracks when a specific scenario's probability modifier recovers.
	// Key: scenario_id, Value: The time the fatigue penalty expires.
	Fatigue map[string]time.Time
}

// NewSimulationState initializes a fresh ledger for a new simulation run.
func NewSimulationState(blueprint *domain.NodeArchetype, startTime time.Time, startTemp float64) *SimulationState {
	state := &SimulationState{
		Blueprint:   blueprint,
		SimTime:     startTime,
		IndoorTempC: startTemp,
		Devices:     make(map[string]*DeviceLedger),
		Actors:      make(map[string]*ActorLedger),
		// Pre-allocate a reasonable capacity for events to save allocations in the hot loop
		ActiveEventIDs: make([]string, 0, 10),
	}

	// Initialize all devices to OFF and no cooldowns
	for _, dev := range blueprint.Devices {
		state.Devices[dev.DeviceID] = &DeviceLedger{
			State: domain.DeviceStateOff,
		}
	}

	// Initialize all actors as free and rested
	for _, actor := range blueprint.Actors {
		state.Actors[actor.ActorID] = &ActorLedger{
			IsBusy:  false,
			Fatigue: make(map[string]time.Time),
		}
	}

	return state
}

// ---------------------------------------------------------
// HELPER METHODS (State Mutators)
// ---------------------------------------------------------

// IsDeviceAvailable checks if a device can be triggered right now.
func (s *SimulationState) IsDeviceAvailable(deviceID string) bool {
	ledger, exists := s.Devices[deviceID]
	if !exists {
		return false
	}
	// It must be OFF (or Standby) and the cooldown must have expired.
	if ledger.State == domain.DeviceStateOn {
		return false
	}
	if s.SimTime.Before(ledger.CooldownUntil) {
		return false
	}
	return true
}

// IsActorAvailable checks if a human is free to start a new scenario.
func (s *SimulationState) IsActorAvailable(actorID string) bool {
	ledger, exists := s.Actors[actorID]
	if !exists {
		return false
	}
	// If the current time is past their BusyUntil time, they are free.
	if ledger.IsBusy && s.SimTime.Before(ledger.BusyUntil) {
		return false
	}
	return true
}
