// lib/profilecompiler/biology.go
package profile

import (
	"github.com/tinywideclouds/go-sim-schema/domain"
)

const (
	BaseEnergyDecay    = 6.0
	BaseHungerDecay    = 8.0
	BaseHygieneDecay   = 5.0
	EnergyPerREMCycle  = 20.0
	LeisurePerREMCycle = 15.0
)

func GenerateBiology(intent PersonaIntent) ([]domain.MeterTemplate, domain.ActionTemplate) {
	meters := []domain.MeterTemplate{
		{MeterID: "energy", Max: 100.0, BaseDecayPerHour: BaseEnergyDecay, Curve: "linear"},
		{MeterID: "hunger", Max: 100.0, BaseDecayPerHour: BaseHungerDecay, Curve: "exponential"},
		{MeterID: "hygiene", Max: 100.0, BaseDecayPerHour: BaseHygieneDecay, Curve: "s_curve"},
	}

	daylightSuppression := domain.BonusCurve{
		ContextKey: "time.hour_float",
		Peak:       16.0, // 3:00 PM
		Width:      5.0,  // Suppresses from roughly 11:00 AM to 9:00 PM
		Magnitude:  -50.0,
	}

	timeForBed := domain.BonusCurve{
		ContextKey: "time.hour_float",
		Peak:       2.0,
		Width:      5.0,
		Magnitude:  60.0,
	}

	sleepAction := domain.ActionTemplate{
		ActionID:      "sleep_in_bed",
		Interruptible: true,
		// INERTIA: It costs 15 points just to decide to go to bed.
		// If they wake up in the middle of the night, this penalty vanishes if they stay in bed!
		InitiationFriction: 15.0,
		Satisfies: map[string]domain.ActionFill{
			"energy": {
				Amount: EnergyPerREMCycle,
				Curve:  "ease_in",
			},
			"leisure": {
				Amount: LeisurePerREMCycle,
				Curve:  "linear",
			},
		},

		Duration: domain.ProbabilityDistribution{
			Type:   domain.DistributionTypeNormal,
			Mean:   "90m",
			StdDev: "15m",
		},
		BonusCurves: []domain.BonusCurve{
			daylightSuppression,
			timeForBed,
		},
	}

	return meters, sleepAction
}
