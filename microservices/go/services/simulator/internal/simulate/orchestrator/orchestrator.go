package orchestrator

import (
	"time"

	"github.com/tinywideclouds/go-maths/pkg/probability"
	"github.com/tinywideclouds/go-power-simulator/internal/simulate/core"
)

// MacroEngine handles Where the actor is (Home/Away/Sleep).
type MacroEngine interface {
	Process(state *core.SimulationState, snapshot core.StateSnapshot, tickDuration time.Duration) ([]core.ActorTickState, []string, []string)
}

// MicroRoutineEngine handles strict sequential tasks.
type MicroRoutineEngine interface {
	Process(state *core.SimulationState, snapshot core.StateSnapshot, tickDuration time.Duration) ([]core.ActorTickState, []string, []string)
}

// MicroUtilityEngine handles emergent biological decisions.
type MicroUtilityEngine interface {
	Process(state *core.SimulationState, snapshot core.StateSnapshot, tickDuration time.Duration) ([]core.ActorTickState, []string, []string)
	GetActorSnapshot(actorID string) core.StateSnapshot
}

type Orchestrator struct {
	stableEngine  MacroEngine
	utilityEngine MicroUtilityEngine
	routineEngine MicroRoutineEngine
	sampler       *probability.DistributionSampler
}

func NewOrchestrator(
	stable MacroEngine,
	utility MicroUtilityEngine,
	routine MicroRoutineEngine,
	sampler *probability.DistributionSampler,
) *Orchestrator {
	return &Orchestrator{
		stableEngine:  stable,
		utilityEngine: utility,
		routineEngine: routine,
		sampler:       sampler,
	}
}

func (o *Orchestrator) Tick(state *core.SimulationState, tickDuration time.Duration, weather core.WeatherProvider, grid core.GridProvider) core.TickResult {
	snapshot := core.BuildEnvironmentSnapshot(state.SimTime, weather)

	snapshot["indoor_temp_c"] = state.IndoorTempC
	snapshot["tank_temp_c"] = state.HotWaterTankC

	o.reapFinishedEvents(state)

	var activeActors []core.ActorTickState
	var anomalies []string
	var debugLogs []string

	// 1. MACRO-BRAIN: Update Where everyone is
	mActors, mAnom, mDebug := o.stableEngine.Process(state, snapshot, tickDuration)
	activeActors = append(activeActors, mActors...)
	anomalies = append(anomalies, mAnom...)
	debugLogs = append(debugLogs, mDebug...)

	// 2. MICRO-BRAIN (Routines)
	rActors, rAnom, rDebug := o.routineEngine.Process(state, snapshot, tickDuration)
	activeActors = append(activeActors, rActors...)
	anomalies = append(anomalies, rAnom...)
	debugLogs = append(debugLogs, rDebug...)

	// 3. MICRO-BRAIN (Utility)
	uActors, uAnom, uDebug := o.utilityEngine.Process(state, snapshot, tickDuration)
	activeActors = append(activeActors, uActors...)
	anomalies = append(anomalies, uAnom...)
	debugLogs = append(debugLogs, uDebug...)

	// 4. AMBIENT SYSTEMS
	ProcessAmbientSystems(state, snapshot, o.sampler)

	// 5. PHYSICS
	physicsResult := core.ProcessPhysics(state, tickDuration, grid, map[string]int{})
	state.ActiveEventIDs = append(activeActorsToIDs(activeActors), physicsResult.ActiveDevices...)

	// run the thermodynamics!
	// Get the live outside temperature from our smart weather mock
	externalTemp := weather.GetTemperature(state.SimTime)

	// Process the heat loss against the insulation, plus any active heating watts
	core.ProcessThermodynamics(state, physicsResult.HeaterWatts, externalTemp, tickDuration)

	allMeters := make(map[string]map[string]float64)
	for _, a := range state.Blueprint.Actors {
		snap := o.utilityEngine.GetActorSnapshot(a.ActorID)
		if snap != nil {
			allMeters[a.ActorID] = map[string]float64{
				"energy":  snap["actor.energy"].(float64),
				"hunger":  snap["actor.hunger"].(float64),
				"hygiene": snap["actor.hygiene"].(float64),
				"leisure": snap["actor.leisure"].(float64),
			}
		}
	}

	return core.TickResult{
		Timestamp:       state.SimTime,
		GridVoltage:     physicsResult.GridVoltage,
		TotalWatts:      physicsResult.TotalWatts,
		TotalColdLiters: physicsResult.ColdLiters,
		TotalHotLiters:  physicsResult.HotLiters,
		IndoorTempC:     state.IndoorTempC,
		ExternalTempC:   externalTemp,
		TankTempC:       state.HotWaterTankC,
		ActiveDevices:   physicsResult.ActiveDevices,
		ActiveActors:    activeActors,
		AllHumanMeters:  allMeters,
		Anomalies:       anomalies,
		DebugLog:        debugLogs,
	}
}

func (o *Orchestrator) reapFinishedEvents(state *core.SimulationState) {
	for id, ev := range state.House.PendingEvents {
		if ev.IsExecuting {
			if state.SimTime.After(ev.GatheringEndsAt) || state.SimTime.Equal(ev.GatheringEndsAt) {
				delete(state.House.PendingEvents, id)
				if ev.DeviceID != "" {
					if lock, ok := state.House.ResourceLocks[ev.DeviceID]; ok && lock == id {
						delete(state.House.ResourceLocks, ev.DeviceID)
					}
				}
				for _, participantID := range ev.Participants {
					if ledger, ok := state.Actors[participantID]; ok {
						if ledger.CurrentCommitment != nil && ledger.CurrentCommitment.ActionID == ev.ActionID {
							ledger.CurrentCommitment = nil
						}
					}
				}
			}
		}
	}
}

func activeActorsToIDs(actors []core.ActorTickState) []string {
	var ids []string
	for _, a := range actors {
		ids = append(ids, a.ActionID)
	}
	return ids
}
