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
	Modifiers []DistributionModifier `yaml:"modifiers"` // Restored to support modifiers.go!
}

// RoutineTemplate is the reusable blueprint of ordered tasks (e.g., "Morning Prep").
type RoutineTemplate struct {
	RoutineID   string   `yaml:"routine_id"`
	Description string   `yaml:"description"`
	Tasks       []string `yaml:"tasks"` // Array of Scenario IDs in priority order
}

// ActorRoutine is the personalized application of a template to a specific actor.
type ActorRoutine struct {
	RoutineID string                  `yaml:"routine_id"`
	Trigger   ProbabilityDistribution `yaml:"trigger"`
	Deadline  ProbabilityDistribution `yaml:"deadline"`
}

// ActorTemplate represents an individual agent with their own daily schedule.
type ActorTemplate struct {
	ActorID  string         `yaml:"actor_id"`
	Type     string         `yaml:"type"` // e.g., "adult", "teenager", "system"
	Routines []ActorRoutine `yaml:"routines"`
}

// DependentActor represents someone dragged into a collective event, including their resistance.
type DependentActor struct {
	ActorID        string  `yaml:"actor_id"`
	FrictionWeight float64 `yaml:"friction_weight"` // 0.0 to 1.0 (Pull on the timeline)
	PatienceLimit  string  `yaml:"patience_limit"`  // Duration before forcing an abort
}

// CollectiveEvent synchronizes multiple actors, negotiating time via friction.
type CollectiveEvent struct {
	EventID         string           `yaml:"event_id"`
	LeadActor       string           `yaml:"lead_actor"`
	DependentActors []DependentActor `yaml:"dependent_actors"`
	Action          string           `yaml:"action"` // What happens when the event fires
}
