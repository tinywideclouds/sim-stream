package domain

import (
	"fmt"
	"gopkg.in/yaml.v3"
)

// DeviceCategory classifies what the appliance does for thermodynamic grouping.
type DeviceCategory int

const (
	DeviceCategoryUnspecified DeviceCategory = iota
	DeviceCategoryCooking
	DeviceCategoryHeating
	DeviceCategoryColdStorage
	DeviceCategoryLighting
	DeviceCategoryWetAppliance
	DeviceCategoryElectronics
	DeviceCategoryBaseLoad
)

// ProfileType determines how the electrical/water load is applied over time.
type ProfileType int

const (
	ProfileTypeUnspecified ProfileType = iota
	ProfileTypeConstant
	ProfileTypeCyclic
	ProfileTypeVariable
)

// ConditionOperator defines the logical comparison for Environment Snapshot rules.
type ConditionOperator int

const (
	ConditionOperatorUnspecified ConditionOperator = iota
	ConditionOperatorEq
	ConditionOperatorNeq
	ConditionOperatorGt
	ConditionOperatorLt
	ConditionOperatorGte
	ConditionOperatorLte
)

// UnmarshalYAML intercepts the parser to convert the human-readable string
// operators in the YAML (like ">") directly into the typed ConditionOperator enum.
func (c *ConditionOperator) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}

	switch s {
	case "==":
		*c = ConditionOperatorEq
	case "!=":
		*c = ConditionOperatorNeq
	case ">":
		*c = ConditionOperatorGt
	case "<":
		*c = ConditionOperatorLt
	case ">=":
		*c = ConditionOperatorGte
	case "<=":
		*c = ConditionOperatorLte
	default:
		return fmt.Errorf("unknown condition operator: %q", s)
	}
	return nil
}

// MarshalYAML ensures the compiler writes the string format to the generated households.
func (c ConditionOperator) MarshalYAML() (interface{}, error) {
	switch c {
	case ConditionOperatorEq:
		return "==", nil
	case ConditionOperatorNeq:
		return "!=", nil
	case ConditionOperatorGt:
		return ">", nil
	case ConditionOperatorLt:
		return "<", nil
	case ConditionOperatorGte:
		return ">=", nil
	case ConditionOperatorLte:
		return "<=", nil
	default:
		return "", fmt.Errorf("unknown condition operator int: %d", c)
	}
}

// DeviceState tracks the physical state of hardware.
type DeviceState int

const (
	DeviceStateUnspecified DeviceState = iota
	DeviceStateOn
	DeviceStateOff
	DeviceStateStandby
)

// TriggerType defines how a scenario or routine is initiated.
type TriggerType int

const (
	TriggerTypeUnspecified TriggerType = iota
	TriggerTypeTimeOfDay
	TriggerTypeEventReaction
)

// ActorState defines the current lifecycle phase of a human or system.
type ActorState int

const (
	ActorStateUnspecified   ActorState = iota
	ActorStateAsleep                   // Will not trigger ambient events
	ActorStateHomeFree                 // Awake, at home, and idle (Will trigger ambient events)
	ActorStateRoutineActive            // Currently executing a sequential task list
	ActorStateAway                     // Out of the house (Simulation paused for this actor)
)

// SharingType defines how multiple actors interact with the same device.
type SharingType string

const (
	SharingFreeRider SharingType = "free_rider" // Joins without extending duration (e.g., watching TV)
	SharingScalable  SharingType = "scalable"   // Extends duration per person (e.g., cooking more food)
)
