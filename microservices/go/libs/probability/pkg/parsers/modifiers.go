// pkg/parsers/modifiers.go
package parsers

import (
	"fmt"
	"strconv"
	"time"

	"github.com/tinywideclouds/go-sim-schema/domain"
)

// StateSnapshot represents the state of the world or an actor at a specific simulation tick.
type StateSnapshot map[string]any

// ApplyModifiers evaluates the snapshot against the distribution's modifiers and returns a shifted copy.
func ApplyModifiers(baseDistribution domain.ProbabilityDistribution, snapshot StateSnapshot) (domain.ProbabilityDistribution, error) {
	shiftedDistribution := baseDistribution

	totalShiftDuration, err := CalculateShiftDuration(baseDistribution.Modifiers, snapshot)
	if err != nil {
		return shiftedDistribution, err
	}

	if totalShiftDuration != 0 {
		if shiftedDistribution.Mean != "" {
			baseMeanDuration, err := time.ParseDuration(shiftedDistribution.Mean)
			if err == nil {
				shiftedDistribution.Mean = (baseMeanDuration + totalShiftDuration).String()
			}
		}
		if shiftedDistribution.Value != "" {
			baseValueDuration, err := time.ParseDuration(shiftedDistribution.Value)
			if err == nil {
				shiftedDistribution.Value = (baseValueDuration + totalShiftDuration).String()
			}
		}
	}

	return shiftedDistribution, nil
}

// CalculateShiftDuration computes the total time shift applied by a set of modifiers based on the current state.
func CalculateShiftDuration(modifiers []domain.DistributionModifier, snapshot StateSnapshot) (time.Duration, error) {
	var cumulativeShift time.Duration

	for _, modifier := range modifiers {
		matched, err := CheckCondition(modifier.Condition, snapshot)
		if err != nil {
			return 0, fmt.Errorf("failed to evaluate condition %q: %w", modifier.Condition.ContextKey, err)
		}

		if matched {
			shiftDuration, err := evaluateSingleModifierShift(modifier, snapshot)
			if err != nil {
				return 0, fmt.Errorf("failed to calculate shift for modifier: %w", err)
			}
			cumulativeShift += shiftDuration
		}
	}

	return cumulativeShift, nil
}

func evaluateSingleModifierShift(modifier domain.DistributionModifier, snapshot StateSnapshot) (time.Duration, error) {
	var totalShift time.Duration

	if modifier.ShiftMean != "" {
		meanShift, err := time.ParseDuration(modifier.ShiftMean)
		if err == nil {
			totalShift += meanShift
		}
	}

	if modifier.ShiftValue != "" {
		valueShift, err := time.ParseDuration(modifier.ShiftValue)
		if err == nil {
			totalShift += valueShift
		}
	}

	if modifier.ProportionalSkew != 0 {
		contextValueInterface, exists := snapshot[modifier.Condition.ContextKey]
		if exists {
			var actualValue float64
			switch parsedValue := contextValueInterface.(type) {
			case float64:
				actualValue = parsedValue
			case int:
				actualValue = float64(parsedValue)
			}

			targetValue, err := strconv.ParseFloat(modifier.Condition.Value, 64)
			if err == nil {
				delta := targetValue - actualValue
				shiftSeconds := delta * modifier.ProportionalSkew
				skewDuration := time.Duration(shiftSeconds * float64(time.Second))

				if modifier.ClampMin != "" {
					minimumDuration, _ := time.ParseDuration(modifier.ClampMin)
					if skewDuration < minimumDuration {
						skewDuration = minimumDuration
					}
				}
				if modifier.ClampMax != "" {
					maximumDuration, _ := time.ParseDuration(modifier.ClampMax)
					if skewDuration > maximumDuration {
						skewDuration = maximumDuration
					}
				}

				totalShift += skewDuration
			}
		}
	}

	return totalShift, nil
}

// CheckCondition evaluates a single EngineCondition against the current context.
func CheckCondition(condition domain.EngineCondition, snapshot StateSnapshot) (bool, error) {
	contextValue, exists := snapshot[condition.ContextKey]
	if !exists {
		return false, nil
	}

	switch parsedValue := contextValue.(type) {
	case float64:
		targetFloat, err := strconv.ParseFloat(condition.Value, 64)
		if err != nil {
			return false, fmt.Errorf("condition value %q is not a valid float", condition.Value)
		}
		return compareFloat(parsedValue, targetFloat, condition.Operator), nil
	case int:
		targetFloat, err := strconv.ParseFloat(condition.Value, 64)
		if err != nil {
			return false, fmt.Errorf("condition value %q is not a valid float", condition.Value)
		}
		return compareFloat(float64(parsedValue), targetFloat, condition.Operator), nil
	case string:
		return compareString(parsedValue, condition.Value, condition.Operator), nil
	case bool:
		targetBool, err := strconv.ParseBool(condition.Value)
		if err != nil {
			return false, fmt.Errorf("condition value %q is not a valid boolean", condition.Value)
		}
		return compareBool(parsedValue, targetBool, condition.Operator), nil
	default:
		return false, fmt.Errorf("unsupported type for context key %q", condition.ContextKey)
	}
}

func compareFloat(actual float64, target float64, operator domain.ConditionOperator) bool {
	switch operator {
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

func compareString(actual string, target string, operator domain.ConditionOperator) bool {
	switch operator {
	case domain.ConditionOperatorEq:
		return actual == target
	case domain.ConditionOperatorNeq:
		return actual != target
	default:
		return false
	}
}

func compareBool(actual bool, target bool, operator domain.ConditionOperator) bool {
	switch operator {
	case domain.ConditionOperatorEq:
		return actual == target
	case domain.ConditionOperatorNeq:
		return actual != target
	default:
		return false
	}
}
