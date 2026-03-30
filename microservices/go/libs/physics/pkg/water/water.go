package physics

import (
	"time"

	"github.com/tinywideclouds/go-sim-schema/domain"
)

// TickWaterLiters calculates the exact volume of cold and hot water consumed during a specific time window.
// It returns (coldLiters, hotLiters).
func TickWaterLiters(profile *domain.WaterProfile, state domain.DeviceState, activeFor time.Duration) (float64, float64) {
	// If the device has no plumbing, or it's not physically ON, it draws no water.
	if profile == nil || state != domain.DeviceStateOn {
		return 0.0, 0.0
	}

	// Convert the active duration into minutes (e.g., 15s = 0.25 minutes)
	minutesActive := activeFor.Minutes()

	// Calculate split liters
	coldLiters := profile.ColdLitersPerMinute * minutesActive
	hotLiters := profile.HotLitersPerMinute * minutesActive

	return coldLiters, hotLiters
}
