// lib/profilecompiler/food.go
package profile

import (
	"github.com/tinywideclouds/go-sim-schema/domain"
)

func GenerateFood(intent PersonaIntent) []domain.ActionTemplate {
	var actions []domain.ActionTemplate

	var cooker string
	for _, app := range intent.Appliances {
		if app == "cooker_1" || app == "stove_1" || app == "microwave_1" {
			cooker = app
			break
		}
	}

	nightSuppression := domain.UtilityBonusCurve{
		ContextKey: "time.hour_float",
		Peak:       3.0, // 3:00 AM
		Width:      1.5,
		Magnitude:  -80.0,
	}

	// 1. THE SNACK
	actions = append(actions, domain.ActionTemplate{
		ActionID:           "grab_snack",
		InitiationFriction: 0.5,
		Satisfies: map[string]domain.ActionFill{
			"hunger": {Amount: 15.0, Curve: "linear"},
			"energy": {Amount: 5.0, Curve: "linear"},
		},
		BonusCurves: []domain.UtilityBonusCurve{nightSuppression},
		Duration: domain.ProbabilityDistribution{
			Type:  domain.DistributionTypeConstant,
			Value: "5m",
		},
	})

	// 2. THE BREAKFAST
	if intent.Preferences.BreakfastType == "cooked" {
		actions = append(actions, domain.ActionTemplate{
			ActionID:           "cook_breakfast",
			DeviceID:           cooker,
			Interruptible:      false,
			InitiationFriction: 25.0,
			Satisfies: map[string]domain.ActionFill{
				"hunger": {Amount: 100.0, Curve: "linear"},
				"energy": {Amount: 15.0, Curve: "linear"},
			},
			Duration: domain.ProbabilityDistribution{
				Type:   domain.DistributionTypeNormal,
				Mean:   "16m",
				StdDev: "4m",
			},
			BonusCurves: []domain.UtilityBonusCurve{
				nightSuppression,
				{
					ContextKey: "time.hour_float",
					Peak:       8.0, // Morning Pull
					Width:      1.5,
					Magnitude:  40.0,
				},
			},
		})
	}

	// 3. THE DINNER
	actions = append(actions, domain.ActionTemplate{
		ActionID:           "cook_dinner",
		DeviceID:           cooker,
		Interruptible:      false,
		InitiationFriction: 20.0, // Hard to start after a long day of work!
		Satisfies: map[string]domain.ActionFill{
			"hunger": {Amount: 100.0, Curve: "linear"},
			"energy": {Amount: 10.0, Curve: "linear"},
		},
		Duration: domain.ProbabilityDistribution{
			Type:   domain.DistributionTypeNormal,
			Mean:   "35m",
			StdDev: "5m",
		},
		BonusCurves: []domain.UtilityBonusCurve{
			nightSuppression,
			{
				ContextKey: "time.hour_float",
				Peak:       19.0, // 7:00 PM Cultural Pull
				Width:      2.5,  // Wider window than breakfast
				Magnitude:  45.0, // Strong pull to get them off the sofa
			},
		},
	})

	return actions
}
