package power_test

import (
	"testing"
	"time"

	"github.com/tinywideclouds/go-sim-physics/pkg/power"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

func TestTickAverageWatts(t *testing.T) {
	kettle := domain.DeviceProfile{
		Type:         domain.ProfileTypeConstant,
		MaxWatts:     3000.0,
		StandbyWatts: 2.0, // A smart kettle with an LED
	}

	tick := 15 * time.Second

	tests := []struct {
		name      string
		state     domain.DeviceState
		activeFor time.Duration
		wantWatts float64
	}{
		{
			name:      "Completely OFF",
			state:     domain.DeviceStateOff,
			activeFor: 0,
			wantWatts: 0.0,
		},
		{
			name:      "In STANDBY for full tick",
			state:     domain.DeviceStateStandby,
			activeFor: 0,
			wantWatts: 2.0,
		},
		{
			name:      "ON for full tick",
			state:     domain.DeviceStateOn,
			activeFor: 15 * time.Second,
			wantWatts: 3000.0,
		},
		{
			name:      "ON for 10s of a 15s tick (Fractional)",
			state:     domain.DeviceStateOn,
			activeFor: 10 * time.Second,
			// (3000W * 10s + 2W * 5s) / 15s = (30000 + 10) / 15 = 2000.666...
			wantWatts: 2000.6666666666667,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := power.TickAverageWatts(kettle, tt.state, tt.activeFor, tick)
			// Small epsilon check for floating point math
			if got < tt.wantWatts-0.001 || got > tt.wantWatts+0.001 {
				t.Errorf("TickAverageWatts() = %v, want %v", got, tt.wantWatts)
			}
		})
	}
}
