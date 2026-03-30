package physics

import (
	"testing"
	"time"

	"github.com/tinywideclouds/go-sim-schema/domain"
)

func TestTickWaterLiters(t *testing.T) {
	tests := []struct {
		name         string
		profile      *domain.WaterProfile
		state        domain.DeviceState
		activeFor    time.Duration
		expectedCold float64
		expectedHot  float64
	}{
		{
			name: "Device is OFF - Zero Water",
			profile: &domain.WaterProfile{
				ColdLitersPerMinute: 10.0,
				HotLitersPerMinute:  5.0,
			},
			state:        domain.DeviceStateOff,
			activeFor:    1 * time.Minute,
			expectedCold: 0.0,
			expectedHot:  0.0,
		},
		{
			name:         "Nil Profile - Zero Water",
			profile:      nil,
			state:        domain.DeviceStateOn,
			activeFor:    1 * time.Minute,
			expectedCold: 0.0,
			expectedHot:  0.0,
		},
		{
			name: "Full Minute - Mixed Water (e.g., Warm Shower)",
			profile: &domain.WaterProfile{
				ColdLitersPerMinute: 4.0,
				HotLitersPerMinute:  6.0,
			},
			state:        domain.DeviceStateOn,
			activeFor:    1 * time.Minute,
			expectedCold: 4.0,
			expectedHot:  6.0,
		},
		{
			name: "15 Second Tick - Mixed Water",
			profile: &domain.WaterProfile{
				ColdLitersPerMinute: 4.0,
				HotLitersPerMinute:  6.0,
			},
			state:        domain.DeviceStateOn,
			activeFor:    15 * time.Second, // 0.25 minutes
			expectedCold: 1.0,              // 4.0 * 0.25
			expectedHot:  1.5,              // 6.0 * 0.25
		},
		{
			name: "Cold Only (e.g., Toilet Flush)",
			profile: &domain.WaterProfile{
				ColdLitersPerMinute: 8.0,
				HotLitersPerMinute:  0.0,
			},
			state:        domain.DeviceStateOn,
			activeFor:    30 * time.Second, // 0.5 minutes
			expectedCold: 4.0,
			expectedHot:  0.0,
		},
		{
			name: "Hot Only (e.g., Hot Tap)",
			profile: &domain.WaterProfile{
				ColdLitersPerMinute: 0.0,
				HotLitersPerMinute:  12.0,
			},
			state:        domain.DeviceStateOn,
			activeFor:    15 * time.Second, // 0.25 minutes
			expectedCold: 0.0,
			expectedHot:  3.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCold, gotHot := TickWaterLiters(tt.profile, tt.state, tt.activeFor)

			// Floating point comparison with a tiny epsilon
			epsilon := 0.000001

			if diff := gotCold - tt.expectedCold; diff > epsilon || diff < -epsilon {
				t.Errorf("TickWaterLiters() gotCold = %v, want %v", gotCold, tt.expectedCold)
			}

			if diff := gotHot - tt.expectedHot; diff > epsilon || diff < -epsilon {
				t.Errorf("TickWaterLiters() gotHot = %v, want %v", gotHot, tt.expectedHot)
			}
		})
	}
}
