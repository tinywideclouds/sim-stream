package thermal_test

import (
	"testing"
	"time"

	"github.com/tinywideclouds/go-sim-physics/pkg/thermal"
)

func TestNextTemperature(t *testing.T) {
	// A standard house: moderate insulation, standard electric heating efficiency
	props := thermal.ThermalProperties{
		InsulationDecayRate: 0.1,
		HeatingEfficiency:   0.001,
	}

	tick15Min := 15 * time.Minute // We test with a longer tick to see larger temperature swings

	tests := []struct {
		name        string
		currentTemp float64
		outsideTemp float64
		heaterWatts float64
		wantTemp    float64
	}{
		{
			name:        "Winter Decay (Heater OFF)",
			currentTemp: 20.0,
			outsideTemp: 0.0, // Freezing outside
			heaterWatts: 0.0,
			// Loss: 0.1 * 20.0 diff * 0.25 hours = 0.5 degrees lost
			wantTemp: 19.5,
		},
		{
			name:        "Winter Heating (Heater ON)",
			currentTemp: 19.5,
			outsideTemp: 0.0,
			heaterWatts: 4000.0, // 4kW heater running
			// Loss: 0.1 * 19.5 * 0.25h = 0.4875 lost
			// Gain: 0.001 * 4000W * 0.25h = 1.0 degree gained
			// Net: 19.5 - 0.4875 + 1.0 = 20.0125
			wantTemp: 20.0125,
		},
		{
			name:        "Equilibrium (No change)",
			currentTemp: 15.0,
			outsideTemp: 15.0, // Same temp inside and outside
			heaterWatts: 0.0,
			wantTemp:    15.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := thermal.NextTemperature(props, tt.currentTemp, tt.outsideTemp, tt.heaterWatts, tick15Min)

			if got < tt.wantTemp-0.0001 || got > tt.wantTemp+0.0001 {
				t.Errorf("NextTemperature() = %v, want %v", got, tt.wantTemp)
			}
		})
	}
}
