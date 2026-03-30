package thermal

import (
	"time"
)

// ThermalProperties defines the physical constants of the Node (house).
type ThermalProperties struct {
	InsulationDecayRate float64 // e.g., 0.1 (loses 10% of the diff to outside per hour)
	HeatingEfficiency   float64 // e.g., 0.001 (degrees C gained per Watt-hour of heating)
}

// NextTemperature calculates the new indoor temperature after a time tick.
func NextTemperature(props ThermalProperties, currentTemp, outsideTemp, heaterWatts float64, tick time.Duration) float64 {
	// Convert our tick duration (e.g., 15s) into a fraction of an hour for the math
	deltaHours := tick.Hours()

	// 1. Calculate Heat Loss to the environment
	// If it's colder outside, heatLoss is positive. If it's hotter outside, heatLoss is negative (house warms up).
	tempDiff := currentTemp - outsideTemp
	heatLoss := props.InsulationDecayRate * tempDiff * deltaHours

	// 2. Calculate Heat Gain from active heaters
	// A 4000W heater running for 15s (0.00416 hours) yields 16.6 Watt-hours of energy.
	wattHours := heaterWatts * deltaHours
	heatGain := props.HeatingEfficiency * wattHours

	// 3. Resolve the new temperature
	return currentTemp - heatLoss + heatGain
}
