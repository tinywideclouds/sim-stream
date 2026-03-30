package parsers_test

import (
	"testing"

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
		context      parsers.EnvironmentSnapshot
		expectedMean string
	}{
		{
			name: "Warm morning (10 degrees) - No shift applied",
			context: parsers.EnvironmentSnapshot{
				"weather.external_temp_c": 10.0,
			},
			expectedMean: "7h0m0s", // time.Duration format
		},
		{
			name: "Freezing morning (2 degrees) - Shift applied",
			context: parsers.EnvironmentSnapshot{
				"weather.external_temp_c": 2.0,
			},
			expectedMean: "7h30m0s",
		},
		{
			name: "Missing context - Fails gracefully, no shift",
			context: parsers.EnvironmentSnapshot{
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

	ctx := parsers.EnvironmentSnapshot{
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
