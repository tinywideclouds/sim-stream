// lib/profilecompiler/models.go
package profile

// Preferences dictate which library templates the compiler selects.
type Preferences struct {
	BreakfastType string `yaml:"breakfast_type"` // e.g., "cooked", "cereal", "none"
	WashRoutine   string `yaml:"wash_routine"`   // e.g., "quick_shower", "long_bath"
}

// PersonaIntent represents the high-level human goals and routines.
// We write this, and the compiler does the math.
type PersonaIntent struct {
	HouseholdID string   `yaml:"household_id"`
	Archetype   string   `yaml:"archetype"`
	Traits      []string `yaml:"traits"` // e.g., "disciplined", "chaotic"
	Schedule    Schedule `yaml:"schedule"`

	Preferences Preferences `yaml:"preferences"` // Lifestyle choices
	Appliances  []string    `yaml:"appliances"`
}

type Schedule struct {
	TargetSleepHours float64 `yaml:"target_sleep_hours"` // How long they biologically need
	BedTime          float64 `yaml:"bed_time"`           // E.g., 23.0
	DinnerTime       float64 `yaml:"dinner_time"`        // E.g., 18.5
	WorkStart        float64 `yaml:"work_start"`         // E.g., 8.5 (08:30)
	WorkDuration     float64 `yaml:"work_duration"`      // E.g., 9.0
}
