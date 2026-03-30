package power

import (
	"time"

	"github.com/tinywideclouds/go-sim-schema/domain"
)

// TickAverageWatts calculates the average power draw of a device over a specific time window.
// It handles fractional active times (e.g., a device that turns off halfway through a tick).
func TickAverageWatts(profile domain.DeviceProfile, state domain.DeviceState, activeFor time.Duration, tickDuration time.Duration) float64 {
	// If it's physically off, it draws nothing.
	if state == domain.DeviceStateOff {
		return 0.0
	}

	// If it's plugged in but idle, it draws phantom/standby load.
	if state == domain.DeviceStateStandby {
		return profile.StandbyWatts
	}

	// Cap the active time to the tick duration (cannot run for 20s inside a 15s window)
	if activeFor > tickDuration {
		activeFor = tickDuration
	}

	// Calculate the ratio of time spent active vs inactive during this tick
	activeRatio := float64(activeFor) / float64(tickDuration)
	inactiveRatio := 1.0 - activeRatio

	// Determine the active draw based on the profile type
	var activeWatts float64
	switch profile.Type {
	case domain.ProfileTypeConstant:
		activeWatts = profile.MaxWatts
	case domain.ProfileTypeCyclic, domain.ProfileTypeVariable:
		// For the MVP, we treat cyclic/variable devices as drawing MaxWatts when strictly "ON".
		// In a post-MVP world, this could call a sub-function with a duty-cycle multiplier.
		activeWatts = profile.MaxWatts
	default:
		activeWatts = profile.MaxWatts
	}

	// The total average draw for the tick window
	return (activeWatts * activeRatio) + (profile.StandbyWatts * inactiveRatio)
}
