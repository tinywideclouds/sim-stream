package domain

// GridTemplate defines the regional physics of the local power grid.
type GridTemplate struct {
	NominalVoltage float64 `yaml:"nominal_voltage"`
	WaveCenter     float64 `yaml:"wave_center"`
	WaveAmplitude  float64 `yaml:"wave_amplitude"`
	PeakHour       float64 `yaml:"peak_hour"`
	JitterMin      float64 `yaml:"jitter_min"`
	JitterMax      float64 `yaml:"jitter_max"`
}

// WaterSystemTemplate defines the physical plumbing and regional water properties for this specific house.
type WaterSystemTemplate struct {
	TankCapacityLiters         float64 `yaml:"tank_capacity_liters"`
	MainsWaterTempCelsius      float64 `yaml:"mains_water_temp_celsius"`
	MaxTankTempCelsius         float64 `yaml:"max_tank_temp_celsius"`
	StandbyTemperatureLossTick float64 `yaml:"standby_temperature_loss_tick"`
}

// NodeArchetype is the root document representing a full house/building simulation.
type NodeArchetype struct {
	ArchetypeID         string               `yaml:"archetype_id"`
	Description         string               `yaml:"description"`
	BaseTempC           float64              `yaml:"base_temp_c"`
	InsulationDecayRate float64              `yaml:"insulation_decay_rate"`
	Grid                *GridTemplate        `yaml:"grid"`
	WaterSystem         *WaterSystemTemplate `yaml:"water_system"`

	// The Physical Environment
	Devices   []DeviceTemplate   `yaml:"devices"`
	Scenarios []ScenarioTemplate `yaml:"scenarios"`

	// The Entities (Humans & Systems)
	Actors []Actor `yaml:"actors"`

	// Routine Blueprints (The Rails)
	RoutineTemplates []RoutineTemplate `yaml:"routine_templates"`
	Alarms           []AlarmTemplate   `yaml:"alarms"`
	CollectiveEvents []CollectiveEvent `yaml:"collective_events"`

	// Utility Blueprints (The Rubber Bands)
	Meters  []MeterTemplate  `yaml:"meters"`
	Actions []ActionTemplate `yaml:"actions"`

	// Overrides
	CalendarEvents []CalendarEvent `yaml:"calendar_events"`
}
