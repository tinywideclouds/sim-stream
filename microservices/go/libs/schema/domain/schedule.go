// domain/schedule.go
package domain

import "time"

// PhaseType defines the macro-state of the actor during a specific block of the day.
type PhaseType string

const (
	PhaseTypeHome  PhaseType = "home"
	PhaseTypeAway  PhaseType = "away"
	PhaseTypeSleep PhaseType = "sleep"
)

// Phase replaces your old DailyPhase.
type Phase struct {
	PhaseID    string         `yaml:"phase_id"`
	Type       string         `yaml:"type"`
	AnchorTime string         `yaml:"anchor_time"`
	Gravity    float64        `yaml:"gravity"`
	Duration   PhaseDuration  `yaml:"duration"`
	Modifiers  PhaseModifiers `yaml:"modifiers"`
}

// PhaseDuration embeds your existing ProbabilityDistribution from behavior.go
type PhaseDuration struct {
	ProbabilityDistribution `yaml:",inline"`
	Flexibility             time.Duration `yaml:"flexibility"` // Automatically parsed from "2h"
}

type PhaseModifierType string

const (
	Continuous PhaseModifierType = "continuous"
	BlockEnd   PhaseModifierType = "block_end"
)

type PhaseModifiers struct {
	Application string                      `yaml:"application"` // "continuous" or "block_end"
	Effects     map[string]ContinuousEffect `yaml:"effects"`
}

type ContinuousEffect struct {
	Amount float64       `yaml:"amount"`
	Over   time.Duration `yaml:"over"`  // e.g. "8h"
	Curve  string        `yaml:"curve"` // e.g. "front_loaded"
}

// CalendarEvent defines overriding days (holidays, weekends) loaded from localized files.
type CalendarEvent struct {
	Date        string `yaml:"date"`     // e.g., "2026-06-02"
	DayType     string `yaml:"day_type"` // e.g., "holiday", "weekend"
	Description string `yaml:"description"`
}
