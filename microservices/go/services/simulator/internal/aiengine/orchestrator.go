package aiengine

import (
	"time"

	"github.com/tinywideclouds/go-power-simulator/internal/engine"
	"github.com/tinywideclouds/go-sim-probability/pkg/generator"
	"github.com/tinywideclouds/go-sim-probability/pkg/parsers"
)

type TickResult struct {
	Timestamp       time.Time
	GridVoltage     float64
	TotalWatts      float64
	TotalColdLiters float64
	TotalHotLiters  float64
	IndoorTempC     float64
	TankTempC       float64
	ActiveDevices   []string
	ActiveActors    []string
	Anomalies       []string
	DebugLog        []string
}

type AIEngine interface {
	Process(state *engine.SimulationState, snapshot parsers.EnvironmentSnapshot, tickDuration time.Duration) ([]string, []string, []string)
}

type Orchestrator struct {
	humanAI AIEngine
	sampler *generator.Sampler
}

func NewOrchestrator(ai AIEngine, sampler *generator.Sampler) *Orchestrator {
	return &Orchestrator{
		humanAI: ai,
		sampler: sampler,
	}
}

func (o *Orchestrator) Tick(state *engine.SimulationState, tickDuration time.Duration, weather engine.WeatherProvider, grid engine.GridProvider) TickResult {
	state.SimTime = state.SimTime.Add(tickDuration)

	snapshot := engine.BuildEnvironmentSnapshot(state.SimTime, weather)
	snapshot["indoor_temp_c"] = state.IndoorTempC
	snapshot["tank_temp_c"] = state.HotWaterTankC

	snapshot["time.hour_float"] = float64(state.SimTime.Hour()) + (float64(state.SimTime.Minute()) / 60.0) + (float64(state.SimTime.Second()) / 3600.0)

	activeHumanActors, anomalies, debugLogs := o.humanAI.Process(state, snapshot, tickDuration)

	activeAmbientActors := engine.ProcessAmbientSystems(state, snapshot, o.sampler)
	activeActors := append(activeHumanActors, activeAmbientActors...)
	physics := engine.ProcessPhysics(state, tickDuration, grid)

	externalTempC := 5.0
	if val, ok := snapshot["weather.external_temp_c"].(float64); ok {
		externalTempC = val
	}
	engine.ProcessThermodynamics(state, physics.HeaterWatts, externalTempC, tickDuration)

	return TickResult{
		Timestamp:       state.SimTime,
		GridVoltage:     physics.GridVoltage,
		TotalWatts:      physics.TotalWatts,
		TotalColdLiters: physics.ColdLiters,
		TotalHotLiters:  physics.HotLiters,
		IndoorTempC:     state.IndoorTempC,
		TankTempC:       state.HotWaterTankC,
		ActiveDevices:   physics.ActiveDevices,
		ActiveActors:    activeActors,
		Anomalies:       anomalies,
		DebugLog:        debugLogs,
	}
}
