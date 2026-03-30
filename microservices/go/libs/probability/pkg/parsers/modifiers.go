package parsers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/tinywideclouds/go-sim-schema/domain"
)

// EnvironmentSnapshot represents the state of the world at a specific simulation tick.
// e.g., {"weather.external_temp_c": 4.5, "time.is_weekend": false}
type EnvironmentSnapshot map[string]any

// ApplyModifiers evaluates the context against the distribution's modifiers.
// It returns a newly calculated ProbabilityDistribution ready for the Sampler.
func ApplyModifiers(base domain.ProbabilityDistribution, ctx EnvironmentSnapshot) (domain.ProbabilityDistribution, error) {
	// Create a copy so we don't mutate the underlying blueprint
	shifted := base

	for _, mod := range base.Modifiers {
		matched, err := CheckCondition(mod.Condition, ctx)
		if err != nil {
			return shifted, fmt.Errorf("failed to evaluate condition %q: %w", mod.Condition.ContextKey, err)
		}

		if matched {
			err = applyShift(&shifted, mod)
			if err != nil {
				return shifted, fmt.Errorf("failed to apply modifier: %w", err)
			}
		}
	}

	return shifted, nil
}

// CheckCondition evaluates a single EngineCondition against the current context.
func CheckCondition(cond domain.EngineCondition, ctx EnvironmentSnapshot) (bool, error) {
	val, exists := ctx[cond.ContextKey]
	if !exists {
		// If the context key isn't present, the condition safely fails rather than panicking.
		return false, nil
	}

	// Simple type assertions for the MVP
	switch v := val.(type) {
	case float64:
		// Convert the condition's string value to a float for comparison
		target, err := strconv.ParseFloat(cond.Value, 64)
		if err != nil {
			return false, fmt.Errorf("cannot compare float context to non-float condition value %q", cond.Value)
		}
		return compareFloat(v, target, cond.Operator), nil

	case bool:
		target, err := strconv.ParseBool(cond.Value)
		if err != nil {
			return false, fmt.Errorf("cannot compare bool context to non-bool condition value %q", cond.Value)
		}
		return compareBool(v, target, cond.Operator), nil

	default:
		return false, fmt.Errorf("unsupported context value type for key %q", cond.ContextKey)
	}
}

// applyShift mutates the passed distribution copy.
// For the MVP, we only handle shifting time durations (e.g., adding 30m to a wake-up time).
func applyShift(dist *domain.ProbabilityDistribution, mod domain.DistributionModifier) error {
	if mod.ShiftMean == "" {
		return nil
	}

	// We assume 'Mean' is formatted as a Go duration, e.g., "7h0m" for 7:00 AM.
	baseMeanDur, err := time.ParseDuration(dist.Mean)
	if err != nil {
		return fmt.Errorf("invalid base mean duration %q", dist.Mean)
	}

	// time.ParseDuration doesn't like a leading "+", so we trim it.
	// It handles "-" natively (e.g., "-30m").
	cleanShift := strings.TrimPrefix(mod.ShiftMean, "+")
	shiftDur, err := time.ParseDuration(cleanShift)
	if err != nil {
		return fmt.Errorf("invalid shift duration %q", mod.ShiftMean)
	}

	// Apply the shift and write it back to the distribution as a string
	newMean := baseMeanDur + shiftDur
	dist.Mean = newMean.String()

	return nil
}

// Helper: compareFloat
func compareFloat(actual, target float64, op domain.ConditionOperator) bool {
	switch op {
	case domain.ConditionOperatorEq:
		return actual == target
	case domain.ConditionOperatorNeq:
		return actual != target
	case domain.ConditionOperatorGt:
		return actual > target
	case domain.ConditionOperatorLt:
		return actual < target
	case domain.ConditionOperatorGte:
		return actual >= target
	case domain.ConditionOperatorLte:
		return actual <= target
	default:
		return false
	}
}

// Helper: compareBool
func compareBool(actual, target bool, op domain.ConditionOperator) bool {
	switch op {
	case domain.ConditionOperatorEq:
		return actual == target
	case domain.ConditionOperatorNeq:
		return actual != target
	default:
		return false
	}
}
