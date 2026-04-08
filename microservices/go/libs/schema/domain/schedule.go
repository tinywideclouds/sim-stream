// domain/schedule.go
package domain

// PhaseType defines the macro-state of the actor during a specific block of the day.
type PhaseType string

const (
	PhaseTypeHome  PhaseType = "home"
	PhaseTypeAway  PhaseType = "away"
	PhaseTypeSleep PhaseType = "sleep"
)

// DailyPhase represents a societal macro-block (the "Rails").
type DailyPhase struct {
	PhaseID        string       `yaml:"phase_id"`
	AnchorTime     string       `yaml:"anchor_time"`     // e.g., "07:00"
	BufferDuration string       `yaml:"buffer_duration"` // e.g., "30m"
	Type           PhaseType    `yaml:"type"`            // "home", "away", "sleep"
	AwayProfile    *AwayProfile `yaml:"away_profile"`    // Only used if Type == "away"
}

// AwayProfile dictates the fuzzy duration and re-entry state for out-of-house blocks.
type AwayProfile struct {
	Duration  ProbabilityDistribution `yaml:"duration"`
	Modifiers map[string]float64      `yaml:"modifiers"` // Flat shifts to meters on return, e.g., "energy": -40.0
}

// CalendarEvent defines overriding days (holidays, weekends) loaded from localized files.
type CalendarEvent struct {
	Date        string `yaml:"date"`     // e.g., "2026-06-02"
	DayType     string `yaml:"day_type"` // e.g., "holiday", "weekend"
	Description string `yaml:"description"`
}
