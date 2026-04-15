package domain

import (
	"github.com/tinywideclouds/go-maths/pkg/probability"
)

// BiologyConfig defines the template bounds for a persona's metabolism.
// Used in the Catalog blueprints to roll variance upon spawning.
type BiologyConfig struct {
	DecayPerHour     probability.SampleSpace `yaml:"decay_per_hour"`
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
	AIModel string `yaml:"ai_model"`

	Routines []ActorRoutine `yaml:"routines"`

	// StartingMeters holds the exact rolled float values for this specific actor at spawn time.
	// (The CatalogPersona uses SampleSpace to generate these).
	StartingMeters map[string]float64 `yaml:"starting_meters"`

	Biology map[string]InstantiatedBiology `yaml:"biology"`

	SoftmaxTemperature float64 `yaml:"softmax_temperature"`

	Phases []Phase `yaml:"phases"`
}
