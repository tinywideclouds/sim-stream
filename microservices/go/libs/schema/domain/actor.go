// domain/actor.go
package domain

// BiologyConfig defines the template bounds for a persona's metabolism.
// We reuse ProbabilityDistribution so the Builder can easily sample it.
type BiologyConfig struct {
	DecayPerHour     ProbabilityDistribution `yaml:"decay_per_hour"`
	PhaseMultipliers map[string]float64      `yaml:"phase_multipliers"`
}

// InstantiatedBiology holds the exact, rolled float values for a specific spawned actor.
type InstantiatedBiology struct {
	DecayPerHour     float64            `yaml:"decay_per_hour"`
	PhaseMultipliers map[string]float64 `yaml:"phase_multipliers"`
}

// Actor defines the human or system entity living in the simulation.
type Actor struct {
	ActorID string `yaml:"actor_id"`
	Type    string `yaml:"type"` // "adult", "child", "system"

	// AIModel dictates the overarching orchestrator ("routine", "utility", or "stable"). Defaults to "routine".
	AIModel string `yaml:"ai_model"`

	// ----------------------------------------------------
	// Routine State (Sequential tasks)
	// ----------------------------------------------------
	Routines []ActorRoutine `yaml:"routines"`

	// ----------------------------------------------------
	// Utility State (Biological/Psychological)
	// ----------------------------------------------------

	// StartingMeters now uses ProbabilityDistribution so the Builder can apply Gaussian variance upon spawning.
	StartingMeters map[string]float64 `yaml:"starting_meters"`

	// Biology holds the actor's unique, instantiated metabolism after the Builder rolls it.
	Biology map[string]InstantiatedBiology `yaml:"biology"`

	// SoftmaxTemperature controls the Roulette Wheel variance.
	SoftmaxTemperature float64 `yaml:"softmax_temperature"`

	// ----------------------------------------------------
	// Stable State (Societal Rails)
	// ----------------------------------------------------
	// Phases define the macro-blocks the actor is expected to follow.
	Phases []Phase `yaml:"phases"`
}
