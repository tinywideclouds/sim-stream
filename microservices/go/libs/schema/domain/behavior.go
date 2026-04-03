package domain

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
}

// ProbabilityDistribution defines how an event or duration is mathematically sampled.
type ProbabilityDistribution struct {
	Type      DistributionType       `yaml:"type"`
	Value     string                 `yaml:"value"`     // Used for Constant
	Timeframe string                 `yaml:"timeframe"` // E.g., "1h" used to scale probabilities
	Mean      string                 `yaml:"mean"`      // Used for Normal
	StdDev    string                 `yaml:"std_dev"`   // Used for Normal
	Min       float64                `yaml:"min"`       // Used for Uniform
	Max       float64                `yaml:"max"`       // Used for Uniform
	Modifiers []DistributionModifier `yaml:"modifiers"` // Supports environmental shifts
}
