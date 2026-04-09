// internal/factory/schema.go
package factory

import (
	"github.com/tinywideclouds/go-sim-schema/domain"
)

// CatalogDevice wraps a physical device with tags for the procedural generator to find.
type CatalogDevice struct {
	ID       string                `yaml:"id"`
	Tags     []string              `yaml:"tags"`
	Template domain.DeviceTemplate `yaml:"template"`
}

// CatalogAction wraps an action, linking it to required hardware tags rather than hardcoded IDs.
type CatalogAction struct {
	ID                string                `yaml:"id"`
	RequiresDeviceTag string                `yaml:"requires_device_tag"`
	Template          domain.ActionTemplate `yaml:"template"`
}

// CatalogPersona represents a demographic archetype with starting biological modifiers.
type CatalogPersona struct {
	ID             string             `yaml:"id"`
	Type           string             `yaml:"type"`
	Traits         []string           `yaml:"traits"`
	StartingMeters map[string]float64 `yaml:"starting_meters"`
	Phases         []domain.Phase     `yaml:"phases"`
}

// CatalogSystem represents ambient or fixed household systems (like thermostats).
type CatalogSystem struct {
	ID        string                    `yaml:"id"`
	Scenarios []domain.ScenarioTemplate `yaml:"scenarios"`
}

type PersonaRequirement struct {
	Type string `yaml:"type"`
	Min  int    `yaml:"min"`
	Max  int    `yaml:"max"`
}

type CatalogWaterSystem struct {
	ID       string                     `yaml:"id"`
	Tags     []string                   `yaml:"tags"`
	Template domain.WaterSystemTemplate `yaml:"template"`
}

type CatalogRoutine struct {
	ID       string                 `yaml:"id"`
	Template domain.RoutineTemplate `yaml:"template"`
}

type CatalogAlarm struct {
	ID       string               `yaml:"id"`
	Template domain.AlarmTemplate `yaml:"template"`
}

// --- NEW CATALOG EVENT STRUCTURES ---

// SelectionBlock dictates how the Builder pulls events from the hat.
type SelectionBlock struct {
	Weight           float64 `yaml:"weight"`
	DefaultFrequency string  `yaml:"default_frequency"`
}

// DependentRule defines the demographic friction for a role.
type DependentRule struct {
	Type           string  `yaml:"type"`
	FrictionWeight float64 `yaml:"friction_weight"`
	PatienceLimit  string  `yaml:"patience_limit"`
}

// CatalogEventTemplate holds the generic rules from the YAML.
type CatalogEventTemplate struct {
	EventID         string                   `yaml:"event_id"`
	Action          string                   `yaml:"action"`
	BaseFragility   float64                  `yaml:"base_fragility"`
	AbortConditions []domain.EngineCondition `yaml:"abort_conditions"`
	LeadRequirement string                   `yaml:"lead_requirement"`
	DependentRules  []DependentRule          `yaml:"dependent_rules"`
}

// CatalogEvent replaces the direct wrap of domain.CollectiveEvent.
type CatalogEvent struct {
	ID        string               `yaml:"id"`
	Selection SelectionBlock       `yaml:"selection"`
	Template  CatalogEventTemplate `yaml:"template"`
}

// CatalogComposition represents the recipe for a procedural household.
type CatalogComposition struct {
	ID                     string               `yaml:"id"`
	Description            string               `yaml:"description"`
	PersonaRequirements    []PersonaRequirement `yaml:"persona_requirements"`
	RequiredDeviceTags     []string             `yaml:"required_device_tags"`
	OptionalDeviceTags     []string             `yaml:"optional_device_tags"`
	SystemIDs              []string             `yaml:"system_ids"`
	RequiredWaterSystemTag string               `yaml:"required_water_system_tag"`

	RoutineIDs []string `yaml:"routine_ids"`
	AlarmIDs   []string `yaml:"alarm_ids"`
	EventIDs   []string `yaml:"event_ids"`
}
