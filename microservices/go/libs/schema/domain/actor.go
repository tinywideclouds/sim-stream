// domain/actor.go
package domain

// ActorTemplate defines the human or system entity living in the simulation.
type ActorTemplate struct {
	ActorID string `yaml:"actor_id"`
	Type    string `yaml:"type"` // "adult", "child", "system"

	// AI Model Selection ("routine" or "utility"). Defaults to "routine".
	AIModel string `yaml:"ai_model"`

	// ----------------------------------------------------
	// V2 (Routine-based) State
	// ----------------------------------------------------
	Routines []ActorRoutine `yaml:"routines"`

	// ----------------------------------------------------
	// V3 (Utility-based) State
	// ----------------------------------------------------
	// StartingMeters defines their initial biological/psychological state (e.g., "hunger": 20)
	StartingMeters map[string]float64 `yaml:"starting_meters"`

	// SoftmaxTemperature controls the Roulette Wheel variance.
	// Lower values (e.g., 0.2) force strict discipline. Higher values (e.g., 2.0) increase chaos.
	SoftmaxTemperature float64 `yaml:"softmax_temperature"`
}
