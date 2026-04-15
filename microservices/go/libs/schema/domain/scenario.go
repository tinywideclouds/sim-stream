package domain

import (
	"github.com/tinywideclouds/go-maths/pkg/probability"
)

// FatigueRule prevents a scenario from triggering continuously.
type FatigueRule struct {
	LockoutDuration probability.SampleSpace `yaml:"lockout_duration"`
}

// ScenarioTrigger defines when an ambient/autonomous scenario fires.
type ScenarioTrigger struct {
	Type           TriggerType       `yaml:"type"`
	BaseConditions []EngineCondition `yaml:"base_conditions"`
	FatigueRule    FatigueRule       `yaml:"fatigue_rule"`
}

// ScenarioAction maps the hardware state changes.
type ScenarioAction struct {
	DeviceID   string                             `yaml:"device_id"`
	State      DeviceState                        `yaml:"state"`
	Parameters map[string]probability.SampleSpace `yaml:"parameters"`
}

// ScenarioTemplate is an independent action (or chain of actions) like boiling a kettle or a thermostat.
type ScenarioTemplate struct {
	ScenarioID string           `yaml:"scenario_id"`
	ActorTags  []string         `yaml:"actor_tags"`
	Trigger    *ScenarioTrigger `yaml:"trigger"`
	Actions    []ScenarioAction `yaml:"actions"`
}
