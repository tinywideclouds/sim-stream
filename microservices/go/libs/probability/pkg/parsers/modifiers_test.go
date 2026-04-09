package parsers_test

import (
	"testing"
	"time"

	"github.com/tinywideclouds/go-sim-probability/pkg/parsers"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

func TestApplyModifiers_WeatherShift(t *testing.T) {
	// 1. Define the base probability: Wake up at 7:00 AM (7h0m)
	baseDist := domain.ProbabilityDistribution{
		Type:   domain.DistributionTypeNormal,
		Mean:   "7h0m",
		StdDev: "15m",
		Modifiers: []domain.DistributionModifier{
			{
				// If temp is strictly less than 5 degrees...
				Condition: domain.EngineCondition{
					ContextKey: "weather.external_temp_c",
					Operator:   domain.ConditionOperatorLt,
					Value:      "5.0",
				},
				// ... shift the mean wake up time by +30 minutes
				ShiftMean: "+30m",
			},
		},
	}

	tests := []struct {
		name         string
		context      parsers.StateSnapshot
		expectedMean string
	}{
		{
			name: "Warm morning (10 degrees) - No shift applied",
			context: parsers.StateSnapshot{
				"weather.external_temp_c": 10.0,
			},
			expectedMean: "7h0m0s", // time.Duration format
		},
		{
			name: "Freezing morning (2 degrees) - Shift applied",
			context: parsers.StateSnapshot{
				"weather.external_temp_c": 2.0,
			},
			expectedMean: "7h30m0s",
		},
		{
			name: "Missing context - Fails gracefully, no shift",
			context: parsers.StateSnapshot{
				// Weather data is missing for some reason
				"time.is_weekend": false,
			},
			expectedMean: "7h0m0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply the rules against the current context
			shiftedDist, err := parsers.ApplyModifiers(baseDist, tt.context)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if shiftedDist.Mean != tt.expectedMean {
				t.Errorf("Expected mean %q, got %q", tt.expectedMean, shiftedDist.Mean)
			}
		})
	}
}

func TestCheckCondition_Booleans(t *testing.T) {
	cond := domain.EngineCondition{
		ContextKey: "time.is_weekend",
		Operator:   domain.ConditionOperatorEq,
		Value:      "true",
	}

	ctx := parsers.StateSnapshot{
		"time.is_weekend": true,
	}

	matched, err := parsers.CheckCondition(cond, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !matched {
		t.Errorf("Expected condition to match true")
	}
}

func TestApplyModifiers_ProportionalSkew(t *testing.T) {
	// Base expectation: Start at 07:00 (7h0m)
	baseDist := domain.ProbabilityDistribution{
		Type: domain.DistributionTypeNormal,
		Mean: "7h0m",
		Modifiers: []domain.DistributionModifier{
			{
				Condition: domain.EngineCondition{
					ContextKey: "metric.value",
					Operator:   domain.ConditionOperatorLt,
					Value:      "30.0",
				},
				// If value is below 30, skew the mean by 60 seconds for every point below 30.
				ProportionalSkew: 60.0,
				// Asymmetric Clamp: It can only delay the event (>= 0m) up to a max of 45 minutes.
				ClampMin: "0m",
				ClampMax: "45m",
			},
		},
	}

	tests := []struct {
		name         string
		contextValue float64
		expectedMean string
	}{
		{
			name:         "Value above threshold (40.0) - No shift applied",
			contextValue: 40.0,
			expectedMean: "7h0m0s",
		},
		{
			name:         "Value exactly on threshold (30.0) - No shift applied",
			contextValue: 30.0,
			expectedMean: "7h0m0s",
		},
		{
			name: "Value slightly below threshold (20.0) - Proportional shift",
			// Delta is 10.0. Skew is 10 * 60s = 600s = 10m.
			contextValue: 20.0,
			expectedMean: "7h10m0s",
		},
		{
			name: "Value far below threshold (0.0) - Hit maximum clamp",
			// Delta is 30.0. Skew is 30 * 60s = 1800s = 30m.
			contextValue: 0.0,
			expectedMean: "7h30m0s",
		},
		{
			name: "Extreme value (-50.0) - Enforce ClampMax",
			// Delta is 80.0. Skew is 80 * 60s = 4800s = 80m. (Clamped to 45m).
			contextValue: -50.0,
			expectedMean: "7h45m0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := parsers.StateSnapshot{
				"metric.value": tt.contextValue,
			}

			shiftedDist, err := parsers.ApplyModifiers(baseDist, ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if shiftedDist.Mean != tt.expectedMean {
				t.Errorf("Expected mean %q, got %q", tt.expectedMean, shiftedDist.Mean)
			}
		})
	}
}

func TestCalculateShiftDuration_ProportionalSkew(t *testing.T) {
	modifiers := []domain.DistributionModifier{
		{
			Condition: domain.EngineCondition{
				ContextKey: "actor.hunger",
				Operator:   domain.ConditionOperatorLt,
				Value:      "30.0",
			},
			ProportionalSkew: 60.0, // 1 unit of delta = 60 seconds of shift
			ClampMin:         "-60m",
			ClampMax:         "60m",
		},
	}

	tests := []struct {
		name                 string
		contextValue         float64
		expectedShiftMinutes float64
	}{
		{
			name:                 "Value above threshold - No shift",
			contextValue:         40.0,
			expectedShiftMinutes: 0.0,
		},
		{
			name:                 "Value slightly below threshold - Proportional shift",
			contextValue:         20.0, // Delta = 10. 10 * 60s = 600s = 10m
			expectedShiftMinutes: 10.0,
		},
		{
			name:                 "Value far below threshold - Hit maximum clamp",
			contextValue:         -50.0, // Delta = 80. 80 * 60s = 4800s = 80m. Clamped to 60m.
			expectedShiftMinutes: 60.0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			snapshot := parsers.StateSnapshot{
				"actor.hunger": tc.contextValue,
			}

			shiftDuration, err := parsers.CalculateShiftDuration(modifiers, snapshot)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			expectedDuration := time.Duration(tc.expectedShiftMinutes * float64(time.Minute))
			if shiftDuration != expectedDuration {
				t.Errorf("Expected shift of %v, got %v", expectedDuration, shiftDuration)
			}
		})
	}
}
