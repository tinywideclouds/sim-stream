package runner

import (
	"log/slog"
	"time"

	"github.com/tinywideclouds/go-power-simulator/internal/simulate/core"
)

type ActorReporter interface {
	AddActorAction(householdID, actorID, actionID string, isShared bool, simTime time.Time) error
}

type PowerReporter interface {
	AddPowerUsage(householdID string, simTime time.Time, totalWatts, indoorTempC, externalTempC, tankTempC float64, activeDevices []string) error
}

type MeterReporter interface {
	AddActorMeters(householdID, actorID string, simTime time.Time, energy, hunger, hygiene, leisure float64) error
}

type TickProvider interface {
	Tick(state *core.SimulationState, tickDuration time.Duration, weather core.WeatherProvider, grid core.GridProvider) core.TickResult
}

type Runner struct {
	Orchestrator  TickProvider
	ActorReporter ActorReporter
	PowerReporter PowerReporter
	MeterReporter MeterReporter
}

func NewRunner(orch TickProvider, actorRep ActorReporter, powerRep PowerReporter, meterRep MeterReporter) *Runner {
	return &Runner{
		Orchestrator:  orch,
		ActorReporter: actorRep,
		PowerReporter: powerRep,
		MeterReporter: meterRep,
	}
}

// Run executes the simulation loop.
func (r *Runner) Run(
	state *core.SimulationState,
	simulationLength time.Duration,
	samplingInterval time.Duration,
	telemetryInterval time.Duration,
	weather core.WeatherProvider,
	grid core.GridProvider,
) error {
	householdID := state.Blueprint.ArchetypeID
	previousActions := make(map[string]core.ActorTickState)

	timeSinceLastTelemetry := telemetryInterval

	for elapsed := time.Duration(0); elapsed < simulationLength; elapsed += samplingInterval {
		res := r.Orchestrator.Tick(state, samplingInterval, weather, grid)
		simTimeStr := res.Timestamp.Format("Mon 15:04")

		shouldLogTelemetry := false
		if timeSinceLastTelemetry >= telemetryInterval {
			shouldLogTelemetry = true
			timeSinceLastTelemetry = 0
		}

		if shouldLogTelemetry {
			if r.PowerReporter != nil {
				err := r.PowerReporter.AddPowerUsage(
					householdID,
					res.Timestamp,
					res.TotalWatts,
					res.IndoorTempC,
					res.ExternalTempC,
					res.TankTempC,
					res.ActiveDevices,
				)
				if err != nil {
					return err
				}
			}

			if r.MeterReporter != nil {
				for actorID, meters := range res.AllHumanMeters {
					err := r.MeterReporter.AddActorMeters(
						householdID,
						actorID,
						res.Timestamp,
						meters["energy"],
						meters["hunger"],
						meters["hygiene"],
						meters["leisure"],
					)
					if err != nil {
						return err
					}
				}
			}
		}

		currentActions := make(map[string]core.ActorTickState)
		for _, act := range res.ActiveActors {
			currentActions[act.ActorID] = act
		}

		for actorID, currentState := range currentActions {
			oldState, exists := previousActions[actorID]
			if !exists || oldState.ActionID != currentState.ActionID {
				if exists && oldState.ActionID != "Idle/Away" {
					slog.Info("ACTOR ACTION ENDED", "actor", actorID, "action", oldState.ActionID, "sim_time", simTimeStr)
				}
				if currentState.ActionID != "Idle/Away" {
					slog.Info("ACTOR ACTION STARTED", "actor", actorID, "action", currentState.ActionID, "is_shared", currentState.IsShared, "sim_time", simTimeStr)
				}
				if r.ActorReporter != nil {
					if err := r.ActorReporter.AddActorAction(householdID, actorID, currentState.ActionID, currentState.IsShared, res.Timestamp); err != nil {
						return err
					}
				}
			}
		}

		for actorID, oldState := range previousActions {
			if _, exists := currentActions[actorID]; !exists {
				if oldState.ActionID != "Idle/Away" {
					slog.Info("ACTOR ACTION ENDED", "actor", actorID, "action", oldState.ActionID, "sim_time", simTimeStr)
					slog.Info("ACTOR ACTION STARTED", "actor", actorID, "action", "Idle/Away", "is_shared", false, "sim_time", simTimeStr)
					if r.ActorReporter != nil {
						if err := r.ActorReporter.AddActorAction(householdID, actorID, "Idle/Away", false, res.Timestamp); err != nil {
							return err
						}
					}
				}
				currentActions[actorID] = core.ActorTickState{ActorID: actorID, ActionID: "Idle/Away", IsShared: false}
			}
		}

		previousActions = currentActions
		timeSinceLastTelemetry += samplingInterval

		// BUGFIX: Strictly advance the simulation state time by the interval
		state.SimTime = state.SimTime.Add(samplingInterval)
	}

	return nil
}
