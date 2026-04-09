package engine

import (
	"math"
	"testing"
	"time"

	"github.com/tinywideclouds/go-sim-schema/domain"
)

type MockGrid struct{}

func (m *MockGrid) NominalVoltage() float64         { return 230.0 }
func (m *MockGrid) LiveVoltage(t time.Time) float64 { return 230.0 }

func TestProcessPhysics_JouleHeating(t *testing.T) {
	blueprint := &domain.NodeArchetype{
		WaterSystem: &domain.WaterSystemTemplate{
			TankCapacityLiters:         150.0,
			MaxTankTempCelsius:         60.0,
			StandbyTemperatureLossTick: 0.002,
		},
		Devices: []domain.DeviceTemplate{
			{
				DeviceID: "boiler_1",
				Taxonomy: domain.DeviceTaxonomy{Category: 6},
				ElectricalProfile: domain.DeviceProfile{
					MaxWatts:     3000.0,
					StandbyWatts: 0.0,
				},
			},
		},
	}

	state := NewSimulationState(blueprint, time.Now())
	state.HotWaterTankC = 50.0

	state.Devices["boiler_1"].State = domain.DeviceStateOn
	state.Devices["boiler_1"].StateEndsAt = state.SimTime.Add(1 * time.Hour)

	tickDuration := 15 * time.Second
	grid := &MockGrid{}
	participantCounts := make(map[string]int)

	_ = ProcessPhysics(state, tickDuration, grid, participantCounts)

	// MATH: 45000 Joules / (150L * 4184) = 0.0717 C
	expectedTemp := 50.0 + (45000.0 / (150.0 * 4184.0)) - 0.002

	if math.Abs(state.HotWaterTankC-expectedTemp) > 0.0001 {
		t.Errorf("Expected Tank Temp %.4f, got %.4f", expectedTemp, state.HotWaterTankC)
	}
}

func TestProcessPhysics_ColdWaterMixing(t *testing.T) {
	blueprint := &domain.NodeArchetype{
		WaterSystem: &domain.WaterSystemTemplate{
			TankCapacityLiters:         150.0,
			MainsWaterTempCelsius:      10.0,
			StandbyTemperatureLossTick: 0.002,
		},
		Devices: []domain.DeviceTemplate{
			{
				DeviceID:          "shower_1",
				Taxonomy:          domain.DeviceTaxonomy{Category: 5},
				ElectricalProfile: domain.DeviceProfile{MaxWatts: 0.0, StandbyWatts: 0.0},
				WaterProfile: &domain.WaterProfile{
					HotLitersPerMinute:  12.0, // Draws 3L per 15s
					ColdLitersPerMinute: 0.0,
				},
			},
		},
	}

	state := NewSimulationState(blueprint, time.Now())
	state.HotWaterTankC = 50.0

	state.Devices["shower_1"].State = domain.DeviceStateOn
	state.Devices["shower_1"].StateEndsAt = state.SimTime.Add(1 * time.Hour)

	tickDuration := 15 * time.Second
	grid := &MockGrid{}
	participantCounts := make(map[string]int)

	result := ProcessPhysics(state, tickDuration, grid, participantCounts)

	if result.HotLiters != 3.0 {
		t.Errorf("Expected 3.0 Hot Liters drawn, got %.2f", result.HotLiters)
	}

	// MATH: ((147 * 50) + (3 * 10)) / 150 = 49.2C
	expectedTemp := 49.2 - 0.002

	if math.Abs(state.HotWaterTankC-expectedTemp) > 0.0001 {
		t.Errorf("Expected Tank Temp %.4f, got %.4f", expectedTemp, state.HotWaterTankC)
	}
}
