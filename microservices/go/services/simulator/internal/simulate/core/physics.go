package core

import (
	"time"

	"github.com/tinywideclouds/go-sim-physics/pkg/thermal"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

// Universal Physics Constants
const SpecificHeatWaterJoules = 4184.0 // Joules required to heat 1 Liter by 1 degree Celsius

type PhysicsResult struct {
	GridVoltage   float64
	TotalWatts    float64
	HeaterWatts   float64
	ColdLiters    float64
	HotLiters     float64
	ActiveDevices []string
}

// ProcessPhysics calculates the hardware states, power draw, and tank thermodynamics.
// participantCounts provides the active number of occupants using each device, allowing for future dynamic load scaling.
func ProcessPhysics(state *SimulationState, tickDuration time.Duration, grid GridProvider, participantCounts map[string]int) PhysicsResult {
	var totalWatts float64
	var spaceHeaterWatts float64
	var waterHeaterWatts float64
	var coldLiters float64
	var hotLiters float64
	var activeDeviceIDs []string

	// 1. Extract House-Specific Plumbing (Fallback to defaults if omitted in YAML)
	tankCapacityLiters := 150.0
	mainsWaterTempCelsius := 10.0
	maxTankTempCelsius := 60.0
	standbyTemperatureLossTick := 0.002

	if state.Blueprint.WaterSystem != nil {
		tankCapacityLiters = state.Blueprint.WaterSystem.TankCapacityLiters
		mainsWaterTempCelsius = state.Blueprint.WaterSystem.MainsWaterTempCelsius
		maxTankTempCelsius = state.Blueprint.WaterSystem.MaxTankTempCelsius
		standbyTemperatureLossTick = state.Blueprint.WaterSystem.StandbyTemperatureLossTick
	}

	// 2. Grid Voltage Calculation
	nominalVoltage := grid.NominalVoltage()
	currentVoltage := grid.LiveVoltage(state.SimTime)
	voltageMultiplier := (currentVoltage * currentVoltage) / (nominalVoltage * nominalVoltage)

	// 3. Process All Devices
	for _, deviceTemplate := range state.Blueprint.Devices {
		ledger := state.Devices[deviceTemplate.DeviceID]

		// Turn off devices whose duration has expired
		if ledger.State == domain.DeviceStateOn && !state.SimTime.Before(ledger.StateEndsAt) {
			ledger.State = domain.DeviceStateStandby
		}

		switch ledger.State {
		case domain.DeviceStateOn:
			activeDeviceIDs = append(activeDeviceIDs, deviceTemplate.DeviceID)

			actualWatts := deviceTemplate.ElectricalProfile.MaxWatts * voltageMultiplier
			totalWatts += actualWatts

			if deviceTemplate.ThermalProfile != nil {
				// E.g., Gas Boiler (80W electrical, 24000W thermal)
				if deviceTemplate.Taxonomy.Category == domain.DeviceCategoryHeating {
					spaceHeaterWatts += deviceTemplate.ThermalProfile.RadiatedWatts
				} else if deviceTemplate.Taxonomy.Category == domain.DeviceCategory(6) { // Water Heating
					waterHeaterWatts += deviceTemplate.ThermalProfile.RadiatedWatts
				}
			} else {
				// Fallback: 100% efficient electric heating (e.g., standard electric radiator)
				if deviceTemplate.Taxonomy.Category == domain.DeviceCategoryHeating {
					spaceHeaterWatts += actualWatts
				} else if deviceTemplate.Taxonomy.Category == domain.DeviceCategory(6) {
					waterHeaterWatts += actualWatts
				}
			}

			if deviceTemplate.Taxonomy.Category == domain.DeviceCategoryHeating {
				spaceHeaterWatts += actualWatts
			} else if deviceTemplate.Taxonomy.Category == domain.DeviceCategory(6) { // Water Heating
				waterHeaterWatts += actualWatts
			}

			if deviceTemplate.WaterProfile != nil {
				minutes := tickDuration.Minutes()
				drawnColdLiters := deviceTemplate.WaterProfile.ColdLitersPerMinute * minutes
				drawnHotLiters := deviceTemplate.WaterProfile.HotLitersPerMinute * minutes

				coldLiters += drawnColdLiters
				hotLiters += drawnHotLiters

				// Tank Cooling Dynamics (Mixing cold mains water)
				if drawnHotLiters > 0 {
					remainingLiters := tankCapacityLiters - drawnHotLiters
					if remainingLiters < 0 {
						remainingLiters = 0
					}

					energyOfRemainingWater := remainingLiters * state.HotWaterTankC
					energyOfNewColdWater := drawnHotLiters * mainsWaterTempCelsius

					state.HotWaterTankC = (energyOfRemainingWater + energyOfNewColdWater) / tankCapacityLiters
				}
			}
		case domain.DeviceStateStandby:
			standbyWatts := deviceTemplate.ElectricalProfile.StandbyWatts * voltageMultiplier
			totalWatts += standbyWatts
		}
	}

	// 4. Tank Heating Dynamics (Joule Heating)
	if waterHeaterWatts > 0 {
		joulesAdded := waterHeaterWatts * tickDuration.Seconds()
		temperatureDeltaCelsius := joulesAdded / (tankCapacityLiters * SpecificHeatWaterJoules)
		state.HotWaterTankC += temperatureDeltaCelsius

		if state.HotWaterTankC > maxTankTempCelsius {
			state.HotWaterTankC = maxTankTempCelsius
		}
	}

	// 5. Passive Tank Heat Loss
	state.HotWaterTankC -= standbyTemperatureLossTick

	return PhysicsResult{
		GridVoltage:   currentVoltage,
		TotalWatts:    totalWatts,
		HeaterWatts:   spaceHeaterWatts,
		ColdLiters:    coldLiters,
		HotLiters:     hotLiters,
		ActiveDevices: activeDeviceIDs,
	}
}

func ProcessThermodynamics(state *SimulationState, heaterWatts float64, outsideTemperatureCelsius float64, tickDuration time.Duration) {
	thermalProperties := thermal.ThermalProperties{
		InsulationDecayRate: state.Blueprint.InsulationDecayRate,
		HeatingEfficiency:   0.001,
	}
	state.IndoorTempC = thermal.NextTemperature(thermalProperties, state.IndoorTempC, outsideTemperatureCelsius, heaterWatts, tickDuration)
}
