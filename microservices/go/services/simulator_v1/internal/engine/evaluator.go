package engine

import (
	"fmt"
	"math"
	"time"

	"github.com/tinywideclouds/go-sim-probability/pkg/generator"
	"github.com/tinywideclouds/go-sim-probability/pkg/parsers"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

// Evaluator holds the random number generator and evaluates scenarios.
type Evaluator struct {
	sampler *generator.Sampler
	tickDur time.Duration
}

// NewEvaluator creates a new rules engine.
func NewEvaluator(sampler *generator.Sampler, tickDuration time.Duration) *Evaluator {
	return &Evaluator{
		sampler: sampler,
		tickDur: tickDuration,
	}
}

// ShouldTrigger determines if a specific scenario fires during the current tick.
func (e *Evaluator) ShouldTrigger(scenario domain.ScenarioTemplate, state *SimulationState, env parsers.EnvironmentSnapshot) (bool, error) {

	// 1. HARD LOCK: Is the Actor currently busy doing something else?
	if !state.IsActorAvailable(scenario.ActorID) {
		return false, nil
	}

	// 2. BASE CONDITIONS: e.g., "Must be a weekday" or "Must be raining"
	for _, cond := range scenario.Trigger.BaseConditions {
		matched, err := parsers.CheckCondition(cond, env)
		if err != nil {
			return false, fmt.Errorf("failed base condition check for %s: %w", scenario.ScenarioID, err)
		}
		if !matched {
			return false, nil // Environment doesn't match, abort
		}
	}

	// 3. FATIGUE/SATIETY: Has the actor done this too recently?
	actorLedger := state.Actors[scenario.ActorID]
	if lockoutExpiry, exists := actorLedger.Fatigue[scenario.ScenarioID]; exists {
		if state.SimTime.Before(lockoutExpiry) {
			// They are still in the hard cooldown period for this specific scenario
			return false, nil
		}
	}

	// 4. SITUATIONAL MODIFIERS: Shift the math based on the weather/season
	shiftedDist, err := parsers.ApplyModifiers(scenario.Trigger.Distribution, env)
	if err != nil {
		return false, fmt.Errorf("failed to apply modifiers: %w", err)
	}

	// 5. EVALUATE THE DICE ROLL
	switch scenario.Trigger.Type {

	case domain.TriggerTypeTimeOfDay:
		// MVP Approach: We calculate the target duration from midnight (e.g., 07h30m).
		// If the current time of day falls inside this tick window, we fire.
		targetDur, err := e.sampler.Duration(shiftedDist)
		if err != nil {
			return false, fmt.Errorf("failed to sample TimeOfDay: %w", err)
		}

		// Find how far into the current day we are
		y, m, d := state.SimTime.Date()
		midnight := time.Date(y, m, d, 0, 0, 0, 0, state.SimTime.Location())
		currentDur := state.SimTime.Sub(midnight)

		// Did we hit the exact target window?
		// (e.g., Target = 07:30:00. SimTime = 07:30:05. TickDur = 15s. This is a MATCH).
		if currentDur >= targetDur && currentDur < targetDur+e.tickDur {
			return true, nil
		}
		return false, nil

	case domain.TriggerTypeEventReaction:
		// 1. Get the base chance from the YAML (e.g., 0.80 for 80%)
		chance, err := e.sampler.Float64(shiftedDist)
		if err != nil {
			return false, fmt.Errorf("failed to sample EventReaction chance: %w", err)
		}

		// 2. TIME-SCALING MATH
		// If the YAML specifies a timeframe (e.g., "1h"), scale the probability down to this specific tick.
		if shiftedDist.Timeframe != "" {
			frameDur, err := time.ParseDuration(shiftedDist.Timeframe)
			if err == nil && frameDur > 0 {
				importMath := true // Ensure "math" is imported at the top of your file!
				_ = importMath

				// P_tick = 1 - (1 - P_frame)^(TickDur / FrameDur)
				exponent := float64(e.tickDur) / float64(frameDur)
				chance = 1.0 - math.Pow(1.0-chance, exponent)
			}
		}

		// 3. Roll a D100 (Uniform distribution between 0.0 and 1.0)
		roll, _ := e.sampler.Float64(domain.ProbabilityDistribution{
			Type: domain.DistributionTypeUniform, Min: 0.0, Max: 1.0,
		})

		// Trigger if the roll falls inside our mathematically scaled chance window
		return roll <= chance, nil

	default:
		return false, nil
	}
}
