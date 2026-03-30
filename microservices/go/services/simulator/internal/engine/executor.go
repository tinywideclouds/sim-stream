package engine

import (
	"fmt"
	"time"

	"github.com/tinywideclouds/go-sim-probability/pkg/generator"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

// Executor manages the step-by-step progression of an Actor's routine.
type Executor struct {
	sampler *generator.Sampler
}

// NewExecutor creates a new routine executor.
func NewExecutor(sampler *generator.Sampler) *Executor {
	return &Executor{sampler: sampler}
}

// AdvanceRoutine checks an actor's progress. It starts the next task, or aborts if the deadline is hit.
func (e *Executor) AdvanceRoutine(state *SimulationState, actorID string, template *domain.RoutineTemplate, deadline time.Time) error {
	actor := state.Actors[actorID]

	// 1. THE BREAKING POINT: Check if we have hit the hard deadline
	if !state.SimTime.Before(deadline) {
		e.terminateRoutine(state, actor)
		return nil
	}

	// 2. THE BUSY LOCK: Is the actor still executing the previous task?
	if state.SimTime.Before(actor.StateEndsAt) {
		return nil // Still busy, let them finish
	}

	// 3. COMPLETION CHECK: Have we finished all tasks in the array?
	if actor.RoutineStepIndex >= len(template.Tasks) {
		e.terminateRoutine(state, actor)
		return nil
	}

	// 4. EXECUTE NEXT TASK: Grab the scenario ID from the template array
	scenarioID := template.Tasks[actor.RoutineStepIndex]
	scenario := e.findScenario(state.Blueprint, scenarioID)
	if scenario == nil {
		return fmt.Errorf("scenario %s not found in blueprint", scenarioID)
	}

	var maxDuration time.Duration

	// Apply all physical actions in this scenario (e.g., Turn on Kettle)
	for _, action := range scenario.Actions {
		// Calculate how long this specific device will run
		durDist, ok := action.Parameters["duration"]
		var actionDur time.Duration
		if ok {
			var err error
			actionDur, err = e.sampler.Duration(durDist)
			if err != nil {
				return fmt.Errorf("failed to sample duration for %s: %w", action.DeviceID, err)
			}
		}

		// Update the Device Ledger to turn it ON
		deviceLedger, exists := state.Devices[action.DeviceID]
		if exists {
			deviceLedger.State = action.State
			deviceLedger.StateEndsAt = state.SimTime.Add(actionDur)
		}

		// Track the longest action to lock the actor
		if actionDur > maxDuration {
			maxDuration = actionDur
		}
	}

	// 5. ADVANCE THE STATE MACHINE
	// Lock the actor for the duration of the task
	actor.StateEndsAt = state.SimTime.Add(maxDuration)

	// Increment the index so the next time they are free, they do the next task
	actor.RoutineStepIndex++

	// Record fatigue (Satiety) so they don't repeat this randomly later in the day
	if scenario.Trigger != nil && scenario.Trigger.FatigueRule.LockoutDuration != "" {
		lockout, _ := time.ParseDuration(scenario.Trigger.FatigueRule.LockoutDuration)
		actor.Satiety[scenarioID] = state.SimTime.Add(lockout)
	}

	return nil
}

// terminateRoutine gracefully (or forcefully) ends a routine and frees the actor.
func (e *Executor) terminateRoutine(state *SimulationState, actor *ActorLedger) {
	actor.CurrentState = domain.ActorStateHomeFree
	actor.CurrentRoutineID = ""
	actor.RoutineStepIndex = -1       // Reset index
	actor.StateEndsAt = state.SimTime // Free immediately
}

// findScenario is a helper to look up the template from the Blueprint.
func (e *Executor) findScenario(bp *domain.NodeArchetype, id string) *domain.ScenarioTemplate {
	for _, s := range bp.Scenarios {
		if s.ScenarioID == id {
			return &s
		}
	}
	return nil
}
