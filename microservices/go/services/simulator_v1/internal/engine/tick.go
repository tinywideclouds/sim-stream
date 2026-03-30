package engine

import (
	"fmt"
	"time"

	"github.com/tinywideclouds/go-sim-physics/pkg/power"
	"github.com/tinywideclouds/go-sim-physics/pkg/thermal"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

// TickResult is the payload generated at the end of every 15-second loop.
// This maps almost exactly to our BigQuery Raw Meter Readings schema.
type TickResult struct {
	SimTime        time.Time
	TotalWatts     float64
	IndoorTempC    float64
	ActiveEventIDs []string
}

// Step advances the simulation by exactly one tick duration.
func Step(state *SimulationState, eval *Evaluator, weather WeatherProvider, tickDur time.Duration) (TickResult, error) {

	// 1. SENSES: Look at the outside world
	env := BuildEnvironmentSnapshot(state.SimTime, weather)

	env["node.indoor_temp_c"] = state.IndoorTempC

	// 2. MAINTENANCE: Reap finished events and enforce cooldowns
	cleanUpFinishedTasks(state)

	// 3. BRAIN: Evaluate Scenarios and trigger new events
	for _, scenario := range state.Blueprint.Scenarios {
		triggered, err := eval.ShouldTrigger(scenario, state, env)
		if err != nil {
			return TickResult{}, fmt.Errorf("evaluation failed for %s: %w", scenario.ScenarioID, err)
		}

		if triggered {
			err := executeScenario(scenario, state, eval, tickDur)
			if err != nil {
				return TickResult{}, fmt.Errorf("failed to execute %s: %w", scenario.ScenarioID, err)
			}
		}
	}

	// 4. PHYSICS (ELECTRICAL): Calculate total load for this tick
	totalWatts := 0.0
	heaterWatts := 0.0

	for _, devTemplate := range state.Blueprint.Devices {
		ledger := state.Devices[devTemplate.DeviceID]

		// How long was the device active during this specific tick window?
		activeFor := tickDur
		if ledger.StateEndsAt.Before(state.SimTime.Add(tickDur)) && ledger.StateEndsAt.After(state.SimTime) {
			// It turns off partway through this tick! Calculate fractional time.
			activeFor = ledger.StateEndsAt.Sub(state.SimTime)
		}

		watts := power.TickAverageWatts(devTemplate.ElectricalProfile, ledger.State, activeFor, tickDur)
		totalWatts += watts

		// Isolate heating watts for the thermodynamic model
		if devTemplate.Taxonomy.Category == domain.DeviceCategoryHeating {
			heaterWatts += watts
		}
	}

	// 5. PHYSICS (THERMAL): Calculate heat loss/gain
	// For MVP, we assume a standard heating efficiency. In V2, this moves to the Archetype.
	thermalProps := thermal.ThermalProperties{
		InsulationDecayRate: state.Blueprint.InsulationDecayRate,
		HeatingEfficiency:   0.001, // 1 degree gained per 1000 Watt-hours
	}

	outsideTemp := 15.0 // Fallback
	if weather != nil {
		outsideTemp = weather.GetTemperature(state.SimTime)
	}

	state.IndoorTempC = thermal.NextTemperature(thermalProps, state.IndoorTempC, outsideTemp, heaterWatts, tickDur)

	// 6. SNAPSHOT: Capture the exact state for our data stream
	// We make a copy of the active event IDs so the slice isn't mutated by the next tick
	currentEvents := make([]string, len(state.ActiveEventIDs))
	copy(currentEvents, state.ActiveEventIDs)

	result := TickResult{
		SimTime:        state.SimTime,
		TotalWatts:     totalWatts,
		IndoorTempC:    state.IndoorTempC,
		ActiveEventIDs: currentEvents,
	}

	// 7. TIME CONTINUES: Advance the clock for the next loop
	state.SimTime = state.SimTime.Add(tickDur)

	return result, nil
}

// ---------------------------------------------------------
// INTERNAL HELPERS
// ---------------------------------------------------------

func cleanUpFinishedTasks(state *SimulationState) {
	var activeEvents []string

	for _, ledger := range state.Devices {
		// If the device's run-time has expired, turn it off and apply cooldown
		if ledger.State == domain.DeviceStateOn && !state.SimTime.Before(ledger.StateEndsAt) {
			ledger.State = domain.DeviceStateOff

			// We'll use a hardcoded 5m cooldown for the MVP if not specified
			cooldown := 5 * time.Minute
			ledger.CooldownUntil = state.SimTime.Add(cooldown)
			ledger.ActiveEventID = ""
		}

		// Rebuild the active events array (removing finished ones)
		if ledger.ActiveEventID != "" {
			activeEvents = append(activeEvents, ledger.ActiveEventID)
		}
	}

	// Deduplicate the active events (since one scenario might trigger 3 devices)
	state.ActiveEventIDs = uniqueStrings(activeEvents)
}

func executeScenario(scenario domain.ScenarioTemplate, state *SimulationState, eval *Evaluator, tickDur time.Duration) error {
	// Generate a deterministic UUID for this event so we can join it in BigQuery later
	eventID := fmt.Sprintf("%s-%d", scenario.ScenarioID, state.SimTime.Unix())

	maxBusyDuration := time.Duration(0)

	for _, action := range scenario.Actions {
		// MVP: We assume the primary parameter is "duration" for all standard actions
		dist, ok := action.Parameters["duration"]
		if !ok {
			// For state-driven devices (like Thermostats), duration is handled differently,
			// but for Kettles and Showers, we need a duration distribution.
			continue
		}

		duration, err := eval.sampler.Duration(dist)
		if err != nil {
			return err
		}

		ledger := state.Devices[action.DeviceID]
		ledger.State = action.State
		ledger.StateEndsAt = state.SimTime.Add(duration)
		ledger.ActiveEventID = eventID

		if duration > maxBusyDuration {
			maxBusyDuration = duration
		}
	}

	// Lock the actor for the duration of the longest action they initiated
	actorLedger := state.Actors[scenario.ActorID]
	actorLedger.IsBusy = true
	actorLedger.BusyUntil = state.SimTime.Add(maxBusyDuration)

	// Apply Fatigue penalty
	if scenario.Trigger.FatigueRule.LockoutDuration != "" {
		lockout, _ := time.ParseDuration(scenario.Trigger.FatigueRule.LockoutDuration)
		actorLedger.Fatigue[scenario.ScenarioID] = state.SimTime.Add(lockout)
	}

	return nil
}

func uniqueStrings(input []string) []string {
	keys := make(map[string]bool)
	var list []string
	for _, entry := range input {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
