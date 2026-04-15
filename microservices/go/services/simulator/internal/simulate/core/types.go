package core

import "time"

// ActorTickState represents the structured output of an AI brain for a single simulation tick.
type ActorTickState struct {
	ActorID  string
	ActionID string
	IsShared bool
	Meters   map[string]float64
}

// TickResult represents the complete snapshot of a single simulation loop.
type TickResult struct {
	Timestamp       time.Time
	GridVoltage     float64
	TotalWatts      float64
	TotalColdLiters float64
	TotalHotLiters  float64
	ExternalTempC   float64
	IndoorTempC     float64
	TankTempC       float64
	ActiveDevices   []string
	ActiveActors    []ActorTickState
	AllHumanMeters  map[string]map[string]float64
	Anomalies       []string
	DebugLog        []string
}
