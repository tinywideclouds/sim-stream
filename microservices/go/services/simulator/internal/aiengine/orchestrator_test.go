package aiengine

import (
	"testing"
	"time"

	"github.com/tinywideclouds/go-power-simulator/internal/engine"
	"github.com/tinywideclouds/go-sim-probability/pkg/generator"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

// MockGrid for deterministic physics testing
type MockGrid struct{}

func (m *MockGrid) NominalVoltage() float64         { return 230.0 }
func (m *MockGrid) LiveVoltage(t time.Time) float64 { return 230.0 }

func TestOrchestratorPipeline(t *testing.T) {
	// 1. Setup minimal state
	blueprint := &domain.NodeArchetype{
		WaterSystem: &domain.WaterSystemTemplate{
			TankCapacityLiters:         150.0,
			MaxTankTempCelsius:         60.0,
			StandbyTemperatureLossTick: 0.0, // Disable standby loss for easier math
		},
		Devices: []domain.DeviceTemplate{},
		Actors:  []domain.ActorTemplate{},
	}
	state := engine.NewSimulationState(blueprint, time.Now())

	// 2. Setup the V2 Routine Engine
	var seed [32]byte
	sampler := generator.NewSampler(seed)
	scheduler := engine.NewScheduler(sampler)
	negotiator := engine.NewNegotiator()
	executor := engine.NewExecutor(sampler)

	routineEngine := NewRoutineEngine(scheduler, negotiator, executor, 4)

	// 3. Setup the Orchestrator Pipeline
	orchestrator := NewOrchestrator(routineEngine, sampler)

	// 4. Run one Tick
	tickDuration := 15 * time.Second
	grid := &MockGrid{}

	result := orchestrator.Tick(state, tickDuration, nil, grid)

	// 5. Verify the pipeline returned a valid, assembled result
	if result.GridVoltage != 230.0 {
		t.Errorf("Expected GridVoltage 230.0, got %.2f", result.GridVoltage)
	}
	if result.TotalWatts != 0.0 {
		t.Errorf("Expected 0.0 Watts, got %.2f", result.TotalWatts)
	}
}
