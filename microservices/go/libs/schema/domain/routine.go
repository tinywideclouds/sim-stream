// domain/routine.go
package domain

// RoutineTemplate represents a rigid list of sequential tasks.
type RoutineTemplate struct {
	RoutineID   string   `yaml:"routine_id"`
	Description string   `yaml:"description"`
	Tasks       []string `yaml:"tasks"`
}

// ActorRoutine maps an actor to a routine with specific fuzzy daily bounds.
type ActorRoutine struct {
	RoutineID string                  `yaml:"routine_id"`
	Trigger   ProbabilityDistribution `yaml:"trigger"`
	Deadline  ProbabilityDistribution `yaml:"deadline"`
}

// CollectiveEvent forces multiple actors to synchronize their routines.
// This is the fully instantiated blueprint version that the engine runs.
type CollectiveEvent struct {
	EventID         string            `yaml:"event_id"`
	Action          string            `yaml:"action"`
	BaseFragility   float64           `yaml:"base_fragility"`
	AbortConditions []EngineCondition `yaml:"abort_conditions"`
	LeadActor       string            `yaml:"lead_actor"`
	DependentActors []DependentActor  `yaml:"dependent_actors"`
}

// DependentActor defines how willing a specific follower is to wait for the leader.
type DependentActor struct {
	ActorID        string  `yaml:"actor_id"`
	FrictionWeight float64 `yaml:"friction_weight"`
	PatienceLimit  string  `yaml:"patience_limit"`
}
