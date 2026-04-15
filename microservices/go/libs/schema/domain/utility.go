package domain

import (
	"github.com/tinywideclouds/go-maths/pkg/geom"
	"github.com/tinywideclouds/go-maths/pkg/probability"
)

// ActionFill defines both the amount to replenish and the mathematical shape of the replenishment.
type ActionFill struct {
	Amount float64        `yaml:"amount"`
	Curve  geom.CurveType `yaml:"curve"` // e.g., "linear", "ease_in", "ease_out", "bell"
}

// UnmarshalYAML allows us to support both the new curve format and the old shorthand flat float format.
func (af *ActionFill) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var flatAmount float64
	if err := unmarshal(&flatAmount); err == nil {
		af.Amount = flatAmount
		af.Curve = geom.Linear
		return nil
	}

	type alias ActionFill
	var fullStruct alias
	if err := unmarshal(&fullStruct); err != nil {
		return err
	}

	af.Amount = fullStruct.Amount
	af.Curve = fullStruct.Curve
	if af.Curve == "" {
		af.Curve = geom.Linear
	}
	return nil
}

// MeterTemplate defines an internal biological or psychological need for actors.
type MeterTemplate struct {
	MeterID          string         `yaml:"meter_id"`
	Max              float64        `yaml:"max"`
	BaseDecayPerHour float64        `yaml:"base_decay_per_hour"`
	Curve            geom.CurveType `yaml:"curve"` // e.g., "linear", "exponential"
}

// SharingProfile defines how multiple actors can use the same device simultaneously.
type SharingProfile struct {
	Type                       string  `yaml:"type,omitempty"` // e.g., "free_rider"
	GatheringWindow            string  `yaml:"gathering_window,omitempty"`
	EnergyMultiplierPerActor   float64 `yaml:"energy_multiplier_per_actor,omitempty"`
	DurationMultiplierPerActor float64 `yaml:"duration_multiplier_per_actor,omitempty"`
	MaxParticipants            int     `yaml:"max_participants,omitempty"`
}

// BonusCurve applies time-of-day or contextual Gaussian scoring bonuses (like "Dinner Time").
type BonusCurve struct {
	ContextKey string         `yaml:"context_key"`
	Curve      geom.CurveType `yaml:"curve"`
	Peak       float64        `yaml:"peak"`
	Width      float64        `yaml:"width"`
	Amount     float64        `yaml:"amount"`
}

// ActionModifier applies a flat mathematical adjustment to an action's score if a condition is met.
type ActionModifier struct {
	Condition EngineCondition `yaml:"condition"`
	Amount    float64         `yaml:"amount"`
}

// ActionTemplate defines a dynamic choice an actor can make.
// ActionTemplate defines a dynamic choice an actor can make.
type ActionTemplate struct {
	ActionID  string                `yaml:"action_id"`
	ActorTags []string              `yaml:"actor_tags"`
	DeviceID  string                `yaml:"device_id,omitempty"`
	Satisfies map[string]ActionFill `yaml:"satisfies,omitempty"`

	// UPGRADED to SampleSpace for mathematical variance during execution!
	Costs    map[string]probability.SampleSpace `yaml:"costs,omitempty"`
	Produces map[string]probability.SampleSpace `yaml:"produces,omitempty"`
	Requires map[string]probability.SampleSpace `yaml:"requires,omitempty"`

	AvailableWhen      []EngineCondition   `yaml:"available_when,omitempty"`
	Modifiers          []ActionModifier    `yaml:"modifiers,omitempty"`
	BonusCurves        []BonusCurve        `yaml:"bonus_curves,omitempty"`
	Interruptible      bool                `yaml:"interruptible"`
	InitiationFriction float64             `yaml:"initiation_friction,omitempty"`
	ExpectedMeters     map[string]float64  `yaml:"expected_meters,omitempty"`
	Sharing            *SharingProfile     `yaml:"sharing,omitempty"`
	Duration           DynamicDistribution `yaml:"duration"`
}
