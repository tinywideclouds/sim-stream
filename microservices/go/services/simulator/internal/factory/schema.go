package factory

import (
	"github.com/tinywideclouds/go-maths/pkg/probability"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

type CatalogDevice struct {
	ID       string                `yaml:"id"`
	Tags     []string              `yaml:"tags"`
	Template domain.DeviceTemplate `yaml:"template"`
}

type CatalogAction struct {
	ID                string                `yaml:"id"`
	RequiresDeviceTag string                `yaml:"requires_device_tag"`
	Template          domain.ActionTemplate `yaml:"template"`
}

type CatalogPersona struct {
	ID             string                             `yaml:"id"`
	Type           string                             `yaml:"type"`
	Traits         []string                           `yaml:"traits"`
	Frequency      int                                `yaml:"frequency"`
	StartingMeters map[string]probability.SampleSpace `yaml:"starting_meters"` // Powered by go-maths
	Biology        map[string]domain.BiologyConfig    `yaml:"biology"`
	Phases         []domain.Phase                     `yaml:"phases"`
}

type CatalogSystem struct {
	ID        string                    `yaml:"id"`
	Scenarios []domain.ScenarioTemplate `yaml:"scenarios"`
}

type PersonaRequirement struct {
	Type            string   `yaml:"type"`
	Min             int      `yaml:"min"`
	Max             int      `yaml:"max"`
	AllowedPrefixes []string `yaml:"allowed_prefixes,omitempty"`
	ExcludePrefixes []string `yaml:"exclude_prefixes,omitempty"`
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

type SelectionBlock struct {
	Weight           float64 `yaml:"weight"`
	DefaultFrequency string  `yaml:"default_frequency"`
}

type DependentRule struct {
	Type           string  `yaml:"type"`
	FrictionWeight float64 `yaml:"friction_weight"`
	PatienceLimit  string  `yaml:"patience_limit"`
}

type CatalogEventTemplate struct {
	EventID         string                   `yaml:"event_id"`
	Action          string                   `yaml:"action"`
	BaseFragility   float64                  `yaml:"base_fragility"`
	AbortConditions []domain.EngineCondition `yaml:"abort_conditions"`
	LeadRequirement string                   `yaml:"lead_requirement"`
	DependentRules  []DependentRule          `yaml:"dependent_rules"`
}

type CatalogEvent struct {
	ID        string               `yaml:"id"`
	Selection SelectionBlock       `yaml:"selection"`
	Template  CatalogEventTemplate `yaml:"template"`
}

type CatalogComposition struct {
	ID                     string               `yaml:"id"`
	Frequency              int                  `yaml:"frequency"`
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
