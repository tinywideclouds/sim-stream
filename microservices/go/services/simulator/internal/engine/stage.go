package engine

import (
	"time"

	"github.com/tinywideclouds/go-sim-schema/domain"
)

// DeviceLedger tracks the real-time physical state of a single appliance.
type DeviceLedger struct {
	State       domain.DeviceState
	StateEndsAt time.Time // When the current task or cooldown finishes
}

// ActorLedger tracks the psychological and physical state of a human or system.
type ActorLedger struct {
	CurrentState     domain.ActorState
	CurrentRoutineID string               // E.g., "morning_prep" (Empty if idle)
	RoutineStepIndex int                  // Which task in the array they are currently doing
	StateEndsAt      time.Time            // When the current task finishes (or when they wake up)
	Satiety          map[string]time.Time // Tracks fatigue lockouts: map[scenario_id]UnlockTime
}

// SimulationState is the master ledger for the entire house at any given millisecond.
type SimulationState struct {
	Blueprint *domain.NodeArchetype
	SimTime   time.Time

	// Thermodynamics & Utilities
	IndoorTempC   float64
	HotWaterTankC float64

	// Entity Memory
	Devices map[string]*DeviceLedger
	Actors  map[string]*ActorLedger

	// Active Logging for CSV/BigQuery Output
	ActiveEventIDs []string
}

// NewSimulationState creates a fresh house ready to be simulated.
func NewSimulationState(blueprint *domain.NodeArchetype, startTime time.Time) *SimulationState {
	state := &SimulationState{
		Blueprint:      blueprint,
		SimTime:        startTime,
		IndoorTempC:    blueprint.BaseTempC,
		HotWaterTankC:  55.0, // Default starting tank temperature
		Devices:        make(map[string]*DeviceLedger),
		Actors:         make(map[string]*ActorLedger),
		ActiveEventIDs: []string{},
	}

	// Initialize all devices to OFF
	for _, dev := range blueprint.Devices {
		state.Devices[dev.DeviceID] = &DeviceLedger{
			State:       domain.DeviceStateStandby,
			StateEndsAt: startTime,
		}
	}

	// Initialize all actors to ASLEEP at the start of the simulation (Midnight)
	for _, actor := range blueprint.Actors {
		state.Actors[actor.ActorID] = &ActorLedger{
			CurrentState:     domain.ActorStateAsleep,
			CurrentRoutineID: "",
			RoutineStepIndex: 0,
			StateEndsAt:      startTime,
			Satiety:          make(map[string]time.Time),
		}
	}

	return state
}
