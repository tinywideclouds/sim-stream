// domain/actor.go
package domain

// ActorTemplate defines the human or system entity living in the simulation.
type ActorTemplate struct {
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
	// StartingMeters defines their initial state (e.g., "hunger": 20)
	StartingMeters map[string]float64 `yaml:"starting_meters"`

	// SoftmaxTemperature controls the Roulette Wheel variance.
	SoftmaxTemperature float64 `yaml:"softmax_temperature"`

	// ----------------------------------------------------
	// Stable State (Societal Rails)
	// ----------------------------------------------------
	// Phases define the macro-blocks the actor is expected to follow.
	Phases []DailyPhase `yaml:"phases"`
}
