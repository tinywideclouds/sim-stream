package domain

// DistributionType defines the statistical curve used for random sampling.
type DistributionType int

const (
	DistributionTypeUnspecified DistributionType = iota
	DistributionTypeNormal
	DistributionTypeUniform
	DistributionTypeConstant
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
	ActorStateAway                     // Physically not in the simulation node (Ghosted)
)
