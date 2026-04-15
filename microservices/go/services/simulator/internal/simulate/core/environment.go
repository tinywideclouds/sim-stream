package core

import (
	"fmt"
	"strconv"
	"time"

	"github.com/tinywideclouds/go-sim-schema/domain"
)

// StateSnapshot represents the state of the world or an actor at a specific simulation tick.
type StateSnapshot map[string]any

// EvaluateCondition checks an environment variable against the domain struct using the typed enum.
func EvaluateCondition(cond domain.EngineCondition, snapshot StateSnapshot) bool {
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

	default:
		return false
	}
}

type WeatherProvider interface {
	GetTemperature(t time.Time) float64
	GetPrecipitation(t time.Time) float64
	GetSolarIrradiance(t time.Time) float64
}

func BuildEnvironmentSnapshot(simTime time.Time, weather WeatherProvider) StateSnapshot {
	snap := make(StateSnapshot, 10)

	snap["time.hour"] = float64(simTime.Hour())
	snap["time.minute"] = float64(simTime.Minute())

	weekday := simTime.Weekday()
	snap["time.is_weekend"] = weekday == time.Saturday || weekday == time.Sunday

	snap["season"] = getSeason(simTime.Month())

	if weather != nil {
		snap["weather.external_temp_c"] = weather.GetTemperature(simTime)
		snap["weather.precipitation_mm"] = weather.GetPrecipitation(simTime)
		snap["weather.solar_lux"] = weather.GetSolarIrradiance(simTime)
	}

	return snap
}

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
