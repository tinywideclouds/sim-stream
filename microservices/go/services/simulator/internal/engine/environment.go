package engine

import (
	"fmt"
	"strconv"
	"time"

	"github.com/tinywideclouds/go-sim-probability/pkg/parsers"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

// EvaluateCondition checks an environment variable against the domain struct using the typed enum.
func EvaluateCondition(cond domain.EngineCondition, snapshot parsers.StateSnapshot) bool {
	val, exists := snapshot[cond.ContextKey]
	if !exists {
		return false
	}

	// Safely attempt float conversion for numeric comparisons
	envValStr := fmt.Sprintf("%v", val)
	envValFloat, err1 := strconv.ParseFloat(envValStr, 64)
	condValFloat, err2 := strconv.ParseFloat(cond.Value, 64)

	isNumeric := err1 == nil && err2 == nil

	switch cond.Operator {
	case domain.ConditionOperatorEq:
		if isNumeric {
			return envValFloat == condValFloat
		}
		return envValStr == cond.Value

	case domain.ConditionOperatorNeq:
		if isNumeric {
			return envValFloat != condValFloat
		}
		return envValStr != cond.Value

	case domain.ConditionOperatorGt:
		if isNumeric {
			return envValFloat > condValFloat
		}
		return false

	case domain.ConditionOperatorLt:
		if isNumeric {
			return envValFloat < condValFloat
		}
		return false

	case domain.ConditionOperatorGte:
		if isNumeric {
			return envValFloat >= condValFloat
		}
		return false

	case domain.ConditionOperatorLte:
		if isNumeric {
			return envValFloat <= condValFloat
		}
		return false

	case domain.ConditionOperatorUnspecified:
		return false

	default:
		return false
	}
}

// WeatherProvider defines how the engine asks about the outside world.
// For the MVP, this will be backed by a struct holding the Milan CSV data in memory.
type WeatherProvider interface {
	GetTemperature(t time.Time) float64
	GetPrecipitation(t time.Time) float64
	GetSolarIrradiance(t time.Time) float64 // Useful later for solar panels
}

// BuildEnvironmentSnapshot constructs the exact state of the world for the current tick.
// This is fed directly into the probability modifiers to shift human behavior.
func BuildEnvironmentSnapshot(simTime time.Time, weather WeatherProvider) parsers.StateSnapshot {

	// Pre-allocate the map to avoid resizing during the hot loop
	snap := make(parsers.StateSnapshot, 10)

	// 1. Time-based attributes
	snap["time.hour"] = float64(simTime.Hour())
	snap["time.minute"] = float64(simTime.Minute())

	weekday := simTime.Weekday()
	snap["time.is_weekend"] = weekday == time.Saturday || weekday == time.Sunday

	// 2. Season calculation (Rough approximation for the modifiers)
	snap["season"] = getSeason(simTime.Month())

	// 3. Weather attributes (from the injected Milan dataset)
	if weather != nil {
		snap["weather.external_temp_c"] = weather.GetTemperature(simTime)
		snap["weather.precipitation_mm"] = weather.GetPrecipitation(simTime)
		snap["weather.solar_lux"] = weather.GetSolarIrradiance(simTime)
	}

	return snap
}

// getSeason is a simple helper to map months to string seasons for YAML conditions.
// (Assuming Northern Hemisphere since our MVP is Milan)
func getSeason(m time.Month) string {
	switch m {
	case time.December, time.January, time.February:
		return "winter"
	case time.March, time.April, time.May:
		return "spring"
	case time.June, time.July, time.August:
		return "summer"
	case time.September, time.October, time.November:
		return "autumn"
	default:
		return "unknown"
	}
}
