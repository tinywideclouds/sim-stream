package micro

import (
	"fmt"
	"time"

	"github.com/tinywideclouds/go-maths/pkg/probability"
	"github.com/tinywideclouds/go-power-simulator/internal/simulate/core"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

type Executor struct {
	sampler *probability.DistributionSampler
}

func NewExecutor(sampler *probability.DistributionSampler) *Executor {
	return &Executor{sampler: sampler}
}

func (e *Executor) AdvanceRoutine(state *core.SimulationState, actorID string, template *domain.RoutineTemplate, deadline time.Time) error {
	actor := state.Actors[actorID]

	if !state.SimTime.Before(deadline) {
		e.terminateRoutine(state, actor)
		return nil
	}

	if state.SimTime.Before(actor.StateEndsAt) {
		return nil
	}

	if actor.RoutineStepIndex >= len(template.Tasks) {
		e.terminateRoutine(state, actor)
		return nil
	}

	scenarioID := template.Tasks[actor.RoutineStepIndex]
	scenario := e.findScenario(state.Blueprint, scenarioID)
	if scenario == nil {
		return fmt.Errorf("scenario %s not found in blueprint", scenarioID)
	}

	var maxDuration time.Duration

	for _, action := range scenario.Actions {
		durDist, ok := action.Parameters["duration"]
		var actionDur time.Duration
		if ok {
			// Pure math duration roll!
			actionDur = e.sampler.SampleDuration(durDist)
		}

		deviceLedger, exists := state.Devices[action.DeviceID]
		if exists {
			deviceLedger.State = action.State
			deviceLedger.StateEndsAt = state.SimTime.Add(actionDur)
		}

		if actionDur > maxDuration {
			maxDuration = actionDur
		}
	}

	actor.StateEndsAt = state.SimTime.Add(maxDuration)
	actor.RoutineStepIndex++

	// Pure math lockout roll
	if scenario.Trigger != nil {
		lockout := e.sampler.SampleDuration(scenario.Trigger.FatigueRule.LockoutDuration)
		if lockout > 0 {
			actor.Satiety[scenarioID] = state.SimTime.Add(lockout)
		}
	}

	return nil
}

func (e *Executor) terminateRoutine(state *core.SimulationState, actor *core.ActorLedger) {
	actor.CurrentState = domain.ActorStateHomeFree
	actor.CurrentRoutineID = ""
	actor.RoutineStepIndex = -1
	actor.StateEndsAt = state.SimTime
}

func (e *Executor) findScenario(bp *domain.NodeArchetype, id string) *domain.ScenarioTemplate {
	for _, s := range bp.Scenarios {
		if s.ScenarioID == id {
			return &s
		}
	}
	return nil
}
