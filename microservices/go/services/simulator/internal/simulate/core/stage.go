package core

import (
	"time"

	"github.com/tinywideclouds/go-sim-schema/domain"
)

// DeviceLedger tracks the real-time physical state of a single appliance.
type DeviceLedger struct {
	State       domain.DeviceState
	StateEndsAt time.Time // When the current task or cooldown finishes
	LockedBy    string
}

// ActorLedger tracks the psychological and physical state of a human or system.
type ActorLedger struct {
	CurrentState      domain.ActorState    // The Micro-State (e.g., Idle, RoutineActive)
	CurrentPhase      domain.PhaseType     // The Macro-State (Home, Away, Sleep) - The Bluesky Engine!
	CurrentRoutineID  string               // E.g., "morning_prep" (Empty if idle)
	RoutineStepIndex  int                  // Which task in the array they are currently doing
	StateEndsAt       time.Time            // When the current task finishes (or when they wake up)
	Satiety           map[string]time.Time // Tracks fatigue lockouts: map[scenario_id]UnlockTime
	CurrentCommitment *Commitment
}

type Commitment struct {
	ActionID  string
	Role      string // "lead" or "participant"
	ExpiresAt time.Time
}

type PendingEvent struct {
	EventID         string
	ActionID        string
	DeviceID        string
	InitiatorID     string
	Participants    []string
	GatheringEndsAt time.Time
	IsExecuting     bool
}

type HouseholdLedger struct {
	PendingEvents  map[string]*PendingEvent
	ResourceLocks  map[string]string
	SystemLockouts map[string]time.Time // ScenarioID -> UnlockTime
}

// SimulationState holds the exact, point-in-time memory of the entire household.
type SimulationState struct {
	Blueprint *domain.NodeArchetype
	SimTime   time.Time

	// Thermodynamics & Utilities
	IndoorTempC   float64
	HotWaterTankC float64

	// Entity Memory
	Devices map[string]*DeviceLedger
	Actors  map[string]*ActorLedger
	House   HouseholdLedger

	// Active Logging for CSV/BigQuery Output
	ActiveEventIDs []string
}

// NewSimulationState creates a fresh house ready to be simulated.
func NewSimulationState(blueprint *domain.NodeArchetype, startTime time.Time) *SimulationState {
	state := &SimulationState{
		Blueprint:     blueprint,
		SimTime:       startTime,
		IndoorTempC:   blueprint.BaseTempC,
		HotWaterTankC: 55.0, // Default starting tank temperature
		Devices:       make(map[string]*DeviceLedger),
		Actors:        make(map[string]*ActorLedger),
		House: HouseholdLedger{
			PendingEvents:  make(map[string]*PendingEvent),
			ResourceLocks:  make(map[string]string),
			SystemLockouts: make(map[string]time.Time),
		},
		ActiveEventIDs: []string{},
	}

	// Initialize all devices to OFF/Standby
	for _, dev := range blueprint.Devices {
		state.Devices[dev.DeviceID] = &DeviceLedger{
			State:       domain.DeviceStateStandby,
			StateEndsAt: startTime,
		}
	}

	// Initialize all actors to ASLEEP at the start of the simulation (Midnight)
	for _, actorTemplate := range blueprint.Actors {
		state.Actors[actorTemplate.ActorID] = &ActorLedger{
			CurrentState:     domain.ActorStateAsleep,
			CurrentPhase:     domain.PhaseTypeSleep, // Initializing the macro-state!
			CurrentRoutineID: "",
			RoutineStepIndex: -1,
			StateEndsAt:      startTime, // Ready immediately
			Satiety:          make(map[string]time.Time),
		}
	}

	return state
}
