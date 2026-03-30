package domain

// FatigueRule prevents an actor from spamming the same scenario.
type FatigueRule struct {
	LockoutDuration   string  `yaml:"lockout_duration"`
	RecoveryDuration  string  `yaml:"recovery_duration"`
	PenaltyMultiplier float64 `yaml:"penalty_multiplier"`
}

// Trigger defines how an ambient scenario is initiated (if not part of a routine).
type Trigger struct {
	Type           TriggerType             `yaml:"type"`
	Distribution   ProbabilityDistribution `yaml:"distribution"`
	BaseConditions []EngineCondition       `yaml:"base_conditions"`
	FatigueRule    FatigueRule             `yaml:"fatigue_rule"`
}

// ScenarioAction is a specific physical interaction with a device.
type ScenarioAction struct {
	DeviceID       string                             `yaml:"device_id"`
	State          DeviceState                        `yaml:"state"`
	DelayFromStart string                             `yaml:"delay_from_start"`
	Parameters     map[string]ProbabilityDistribution `yaml:"parameters"`
}

// ScenarioTemplate is an isolated behavior unit (e.g., "Make Tea", "Shower").
type ScenarioTemplate struct {
	ScenarioID string           `yaml:"scenario_id"`
	ActorTags  []string         `yaml:"actor_tags"` // e.g., ["adult", "teen"] for ambient matching
	Trigger    *Trigger         `yaml:"trigger"`    // Pointer: Nil if executed via a Routine Task
	Actions    []ScenarioAction `yaml:"actions"`
}

type GridTemplate struct {
	NominalVoltage float64 `yaml:"nominal_voltage"`
	WaveCenter     float64 `yaml:"wave_center"`
	WaveAmplitude  float64 `yaml:"wave_amplitude"`
	PeakHour       float64 `yaml:"peak_hour"`
	JitterMin      float64 `yaml:"jitter_min"`
	JitterMax      float64 `yaml:"jitter_max"`
}

// NodeArchetype is the root document representing a full house/building simulation.
type NodeArchetype struct {
	ArchetypeID         string             `yaml:"archetype_id"`
	Description         string             `yaml:"description"`
	BaseTempC           float64            `yaml:"base_temp_c"`
	InsulationDecayRate float64            `yaml:"insulation_decay_rate"`
	Actors              []ActorTemplate    `yaml:"actors"`
	Devices             []DeviceTemplate   `yaml:"devices"`
	RoutineTemplates    []RoutineTemplate  `yaml:"routine_templates"`
	Scenarios           []ScenarioTemplate `yaml:"scenarios"`
	CollectiveEvents    []CollectiveEvent  `yaml:"collective_events"`
	Grid                *GridTemplate      `yaml:"grid"`
}
