package orchestrator

import (
	"time"

	"github.com/tinywideclouds/go-maths/pkg/probability"
	"github.com/tinywideclouds/go-power-simulator/internal/simulate/core"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

// ProcessAmbientSystems evaluates environmental scenarios (like thermostats) and locks devices autonomously.
func ProcessAmbientSystems(state *core.SimulationState, snapshot core.StateSnapshot, sampler *probability.DistributionSampler) []string {
	var activeSystems []string

	for _, scenario := range state.Blueprint.Scenarios {
		if scenario.Trigger == nil || scenario.Trigger.Type != domain.TriggerTypeEventReaction {
			continue
		}

		// 1. Check if the system is fatigued (locked out)
		if unlockTime, exists := state.House.SystemLockouts[scenario.ScenarioID]; exists {
			if !state.SimTime.After(unlockTime) {
				continue // Still locked out, skip to next scenario
			}
		}

		// 2. Check if environmental conditions are met
		conditionsMet := true
		for _, condition := range scenario.Trigger.BaseConditions {
			if !core.EvaluateCondition(condition, snapshot) {
				conditionsMet = false
				break
			}
		}

		if conditionsMet {
			var maximumDuration time.Duration

			for _, action := range scenario.Actions {
				durationDistribution, hasDuration := action.Parameters["duration"]
				var actionDuration time.Duration
				if hasDuration {
					// Pure math sampling!
					actionDuration = sampler.SampleDuration(durationDistribution)
				}

				if deviceLedger, exists := state.Devices[action.DeviceID]; exists {
					deviceLedger.State = action.State
					deviceLedger.StateEndsAt = state.SimTime.Add(actionDuration)
				}
				if actionDuration > maximumDuration {
					maximumDuration = actionDuration
				}
			}

			// Sample the fatigue lockout using pure maths
			lockout := sampler.SampleDuration(scenario.Trigger.FatigueRule.LockoutDuration)
			if lockout > 0 {
				state.House.SystemLockouts[scenario.ScenarioID] = state.SimTime.Add(lockout)
			}

			activeSystems = append(activeSystems, "system:"+scenario.ScenarioID)
		}
	}

	return activeSystems
}
