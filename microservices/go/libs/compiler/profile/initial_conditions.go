// lib/profilecompiler/initial_conditions.go
package profile

import (
	"math"
)

const (
	StrategyAlgebraic = "algebraic"
	StrategyBurnIn    = "burn-in"
)

// GenerateInitialConditions calculates the 00:00 (Midnight) state for the actor.
func GenerateInitialConditions(intent PersonaIntent, strategy string) map[string]float64 {
	startState := make(map[string]float64)

	if strategy == StrategyBurnIn {
		// Burn-in is beautifully simple. We spawn them completely average.
		// We rely entirely on the engine's physics and our YAML topography
		// to pull them into a natural orbit over the first 48 simulated hours.
		startState["energy"] = 30.0
		startState["hunger"] = 96.0
		startState["hygiene"] = 90.0
		startState["work_duty"] = 100.0
		startState["leisure"] = 80.0
		return startState
	}

	// Strategy: Algebraic
	// We assume a perfect day happened yesterday, and calculate exactly where
	// the meters would be at exactly 24.0 (Midnight).

	// 1. HUNGER: Decays from Dinner
	hoursSinceDinner := 24.0 - intent.Schedule.DinnerTime
	hungerDecay := BaseHungerDecay * hoursSinceDinner
	startState["hunger"] = math.Max(0.0, 100.0-hungerDecay)

	// 2. WORK DUTY: Decays from end of shift
	// Work fills duty to 100. It decays at 12/hr while at home.
	workEndTime := intent.Schedule.WorkStart + intent.Schedule.WorkDuration
	if workEndTime < 24.0 {
		hoursSinceWork := 24.0 - workEndTime
		dutyDecay := 12.0 * hoursSinceWork
		startState["work_duty"] = math.Max(0.0, 100.0-dutyDecay)
	} else {
		startState["work_duty"] = 100.0 // Night shift edge cases
	}

	// 3. ENERGY: Filling since BedTime
	// We calculate how fast sleep fills energy per hour based on our Biology module
	totalDailyEnergyLoss := 24.0 * BaseEnergyDecay
	targetEnergyReplenish := totalDailyEnergyLoss
	fillPerHour := targetEnergyReplenish / intent.Schedule.TargetSleepHours

	if intent.Schedule.BedTime < 24.0 {
		// They went to bed before midnight (e.g., 23.0).
		// By midnight, they have been sleeping for 1 hour.
		hoursSlept := 24.0 - intent.Schedule.BedTime
		// We assume they went to bed exhausted (near 0) for the algebraic baseline
		startState["energy"] = math.Min(100.0, fillPerHour*hoursSlept)
	} else {
		// The Night Owl: They are still awake at midnight!
		// Assuming they woke up at (BedTime - 24 + TargetSleep), we just dock them some energy.
		// For simplicity in the baseline, we spawn them very tired.
		startState["energy"] = 15.0
	}

	// 4. HYGIENE: Decays from a presumed morning shower
	// We haven't parameterized shower time yet, so we assume 07:30 (7.5) yesterday.
	hoursSinceShower := 24.0 + 0.0 - 7.5 // 16.5 hours ago
	hygDecay := BaseHygieneDecay * hoursSinceShower
	startState["hygiene"] = math.Max(0.0, 100.0-hygDecay)

	return startState
}
