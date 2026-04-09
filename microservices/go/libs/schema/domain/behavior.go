package domain

import "gopkg.in/yaml.v3"

// EngineCondition evaluates an environmental variable against a target.
type EngineCondition struct {
	ContextKey string            `yaml:"context_key"`
	Operator   ConditionOperator `yaml:"operator"`
	Value      string            `yaml:"value"`
}

// DistributionModifier shifts the math based on real-time engine conditions.
type DistributionModifier struct {
	Condition  EngineCondition `yaml:"condition"`
	ShiftMean  string          `yaml:"shift_mean"`
	ShiftValue string          `yaml:"shift_value"`

	// Asymmetrical Skew Mathematics
	ProportionalSkew float64 `yaml:"proportional_skew"` // Multiplier against the delta between the context value and the condition value
	ClampMin         string  `yaml:"clamp_min"`         // e.g., "0m" forces a strictly positive (right-tailed) skew
	ClampMax         string  `yaml:"clamp_max"`         // e.g., "45m" limits the maximum stretch
}

// ProbabilityDistribution defines how an event or duration is mathematically sampled.
// In domain/behavior.go
type ProbabilityDistribution struct {
	Type      DistributionType       `yaml:"type"`
	Value     string                 `yaml:"value,omitempty"`
	Timeframe string                 `yaml:"timeframe,omitempty"`
	Mean      string                 `yaml:"mean,omitempty"`
	StdDev    string                 `yaml:"std_dev,omitempty"`
	Min       float64                `yaml:"min,omitempty"`
	Max       float64                `yaml:"max,omitempty"`
	Modifiers []DistributionModifier `yaml:"modifiers,omitempty"`
}

// UnmarshalYAML intercepts the parser to convert the human-readable string
// operators in the YAML (like ">") directly into the typed ConditionOperator enum.
func (c *ConditionOperator) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}

	switch s {
	case "==":
		*c = ConditionOperatorEq
	case "!=":
		*c = ConditionOperatorNeq
	case ">":
		*c = ConditionOperatorGt
	case "<":
		*c = ConditionOperatorLt
	case ">=":
		*c = ConditionOperatorGte
	case "<=":
		*c = ConditionOperatorLte
	default:
		*c = ConditionOperatorUnspecified
	}
	return nil
}
