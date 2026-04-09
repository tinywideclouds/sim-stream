// lib/profilecompiler/tuning.go
package profile

import (
	"strings"
)

// TuningProfile holds the mathematical coefficients that dictate an actor's discipline.
type TuningProfile struct {
	// SoftmaxTemperature controls the Roulette Wheel variance.
	// Low (e.g., 0.1) = Highly disciplined, almost always picks the highest score.
	// High (e.g., 2.0) = Chaotic, frequently picks lower-scoring distractions.
	// (Note: To be added to the V3 Engine ActorTemplate schema prior to simulation)
	SoftmaxTemperature float64

	// PressureMultiplier scales the magnitude of Gaussian commitment curves.
	// Disciplined actors feel the "pull" of work more intensely.
	PressureMultiplier float64
}

// GenerateTuning evaluates a list of string traits to build a mathematical profile.
func GenerateTuning(traits []string) TuningProfile {
	// Default to an average, somewhat balanced human
	profile := TuningProfile{
		SoftmaxTemperature: 1.0,
		PressureMultiplier: 1.0,
	}

	// Apply Trait Modifiers
	// In the future, this could be a complex weighted matrix. For now, we look for keywords.
	for _, trait := range traits {
		t := strings.ToLower(trait)
		switch t {
		case "disciplined", "punctual":
			profile.SoftmaxTemperature = 0.2 // Ruthlessly picks the highest utility
			profile.PressureMultiplier = 1.5 // Feels work deadlines 50% stronger
		case "chaotic", "lazy":
			profile.SoftmaxTemperature = 2.5 // Flattens probabilities; easily distracted
			profile.PressureMultiplier = 0.7 // Work deadlines don't feel urgent until the last second
		case "anxious":
			// Anxious people might have normal variance, but they feel deadlines intensely
			profile.PressureMultiplier = 2.0
		}
	}

	return profile
}
