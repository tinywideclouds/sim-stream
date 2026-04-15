package domain

import "time"

// PhaseType defines the macro-state of the actor during a specific block of the day.
type PhaseType string

const (
	PhaseTypeHome  PhaseType = "home"
	PhaseTypeAway  PhaseType = "away"
	PhaseTypeSleep PhaseType = "sleep"
)

// MeterEffect defines how a biological meter changes during a block.
type MeterEffect struct {
	Amount float64       `yaml:"amount"`
	Over   time.Duration `yaml:"over,omitempty"`  // The baseline time expected for this drain (e.g., "8h")
	Curve  string        `yaml:"curve,omitempty"` // e.g., "linear", "front_loaded"
}

// PhaseBlock is the atomic unit of time inside a phase.
// It allows an 'Away' phase to consist of multiple probabilistic sub-blocks (e.g. Commute -> Work -> Pub).
type PhaseBlock struct {
	BlockID     string                 `yaml:"block_id"`
	Probability float64                `yaml:"probability"` // 1.0 = always happens, 0.6 = 60% chance
	Duration    DynamicDistribution    `yaml:"duration"`    // Uses the new wrapper around go-maths SampleSpace
	Modifiers   map[string]MeterEffect `yaml:"modifiers"`   // Flattened modifiers map directly attached to the block
}

// Phase represents a major daily macro-state transition, acting as an itinerary container.
type Phase struct {
	PhaseID    string       `yaml:"phase_id"`
	Type       PhaseType    `yaml:"type"` // "home", "away", "sleep"
	AnchorTime string       `yaml:"anchor_time"`
	Gravity    float64      `yaml:"gravity"`
	Blocks     []PhaseBlock `yaml:"blocks"` // All durations and effects live in here now
}

// CalendarEvent defines overriding days (holidays, weekends) loaded from localized files.
type CalendarEvent struct {
	Date        string `yaml:"date"` // e.g. "2026-12-25" or "Saturday"
	Description string `yaml:"description"`
	Type        string `yaml:"type"` // "holiday", "weekend"
}

// AlarmTemplate defines a specific hard trigger for an actor to wake up or change state.
// Powered by DynamicDistribution to allow fuzzy alarm times (e.g., hitting snooze).
type AlarmTemplate struct {
	AlarmID   string              `yaml:"alarm_id"`
	ActorTags []string            `yaml:"actor_tags,omitempty"`
	Time      DynamicDistribution `yaml:"time"`
}
