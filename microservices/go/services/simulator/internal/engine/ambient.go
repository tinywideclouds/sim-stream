package engine

import (
	"time"

	"github.com/tinywideclouds/go-sim-probability/pkg/generator"
	"github.com/tinywideclouds/go-sim-probability/pkg/parsers"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

// ProcessAmbientSystems evaluates environmental scenarios (like thermostats) and locks devices autonomously.
// It returns a slice of active "system:scenario" strings for the CSV logger.
func ProcessAmbientSystems(state *SimulationState, snapshot parsers.StateSnapshot, sampler *generator.Sampler) []string {
	var activeActors []string

	for _, scenario := range state.Blueprint.Scenarios {
		if scenario.Trigger == nil || scenario.Trigger.Type != domain.TriggerTypeEventReaction {
			continue
		}

		conditionsMet := true
		for _, condition := range scenario.Trigger.BaseConditions {
			pass, err := parsers.CheckCondition(condition, snapshot)
			if err != nil || !pass {
				conditionsMet = false
				break
			}
		}

		if conditionsMet {
			var triggeringActor *ActorLedger
			var triggeringActorID string

			// Find an available system actor that isn't currently locked out by fatigue
			for _, tag := range scenario.ActorTags {
				for _, actorTemplate := range state.Blueprint.Actors {
					if actorTemplate.Type == tag {
						ledger := state.Actors[actorTemplate.ActorID]
						if state.SimTime.After(ledger.Satiety[scenario.ScenarioID]) {
							triggeringActor = ledger
							triggeringActorID = actorTemplate.ActorID
							break
						}
					}
				}
				if triggeringActor != nil {
					break
				}
			}

			// Execute the ambient actions
			if triggeringActor != nil {
				var maximumDuration time.Duration
				for _, action := range scenario.Actions {
					durationDistribution, hasDuration := action.Parameters["duration"]
					var actionDuration time.Duration
					if hasDuration {
						actionDuration, _ = sampler.Duration(durationDistribution)
					}

					if deviceLedger, exists := state.Devices[action.DeviceID]; exists {
						deviceLedger.State = action.State
						deviceLedger.StateEndsAt = state.SimTime.Add(actionDuration)
					}
					if actionDuration > maximumDuration {
						maximumDuration = actionDuration
					}
				}

				// Apply Fatigue Lockout to prevent rapid-cycling (e.g., a fridge compressor turning on every single tick)
				if scenario.Trigger.FatigueRule.LockoutDuration != "" {
					lockout, _ := time.ParseDuration(scenario.Trigger.FatigueRule.LockoutDuration)
					triggeringActor.Satiety[scenario.ScenarioID] = state.SimTime.Add(lockout)
				}

				activeActors = append(activeActors, triggeringActorID+":"+scenario.ScenarioID)
			}
		}
	}

	return activeActors
}
