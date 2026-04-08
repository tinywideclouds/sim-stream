// domain/utility.go
package domain

// ActionFill defines both the amount to replenish and the mathematical shape of the replenishment.
type ActionFill struct {
	Amount float64 `yaml:"amount"`
	Curve  string  `yaml:"curve"` // e.g., "linear", "ease_in", "ease_out", "bell"
}

// UnmarshalYAML allows us to support both the new curve format and the old shorthand float format.
func (af *ActionFill) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var flatAmount float64
	if err := unmarshal(&flatAmount); err == nil {
		af.Amount = flatAmount
		af.Curve = "linear"
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
		af.Curve = "linear"
	}
	return nil
}

// MeterTemplate defines an internal biological or psychological need for actors.
type MeterTemplate struct {
	MeterID          string  `yaml:"meter_id"`
	Max              float64 `yaml:"max"`
	BaseDecayPerHour float64 `yaml:"base_decay_per_hour"`
	Curve            string  `yaml:"curve"` // e.g., "linear", "exponential"
}

// AlarmTemplate defines a "signal" that fires at a specific time, pushing context to the snapshot.
type AlarmTemplate struct {
	AlarmID  string `yaml:"alarm_id"`
	Time     string `yaml:"time"`     // e.g., "07:00"
	Duration string `yaml:"duration"` // e.g., "45m"
}

// UtilityModifier applies a flat bonus based on a discrete context state (like an alarm ringing).
type UtilityModifier struct {
	Condition EngineCondition `yaml:"condition"`
	Boost     float64         `yaml:"boost"`
}

// UtilityBonusCurve applies a smooth bell-curve bonus based on a sliding context value (like time of day).
type UtilityBonusCurve struct {
	ContextKey string  `yaml:"context_key"` // e.g., "time.hour"
	Peak       float64 `yaml:"peak"`
	Width      float64 `yaml:"width"`
	Magnitude  float64 `yaml:"magnitude"`
}

// ActionTemplate defines a dynamic choice an actor can make.
type ActionTemplate struct {
	ActionID  string   `yaml:"action_id"`
	ActorTags []string `yaml:"actor_tags"`
	DeviceID  string   `yaml:"device_id"`

	Satisfies map[string]ActionFill `yaml:"satisfies"`

	Costs    map[string]float64 `yaml:"costs"`
	Produces map[string]float64 `yaml:"produces"`
	Requires map[string]float64 `yaml:"requires"`

	AvailableWhen []EngineCondition   `yaml:"available_when"`
	Modifiers     []UtilityModifier   `yaml:"modifiers"`
	BonusCurves   []UtilityBonusCurve `yaml:"bonus_curves"`
	Interruptible bool                `yaml:"interruptible"`

	InitiationFriction float64 `yaml:"initiation_friction"`

	ExpectedMeters map[string]float64 `yaml:"expected_meters"`

	Duration ProbabilityDistribution `yaml:"duration"`
}
