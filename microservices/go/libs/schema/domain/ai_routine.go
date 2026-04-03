package domain

// RoutineTemplate represents a V2 rigid list of sequential tasks.
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

// CollectiveEvent forces multiple V2 actors to synchronize their routines.
type CollectiveEvent struct {
	EventID         string           `yaml:"event_id"`
	LeadActor       string           `yaml:"lead_actor"`
	DependentActors []DependentActor `yaml:"dependent_actors"`
	Action          string           `yaml:"action"`
}

// DependentActor defines how willing a follower is to wait for the leader.
type DependentActor struct {
	ActorID        string  `yaml:"actor_id"`
	FrictionWeight float64 `yaml:"friction_weight"`
	PatienceLimit  string  `yaml:"patience_limit"`
}
