// lib/profilecompiler/hygiene.go
package profile

import (
	"github.com/tinywideclouds/go-sim-schema/domain"
)

func GenerateHygiene(intent PersonaIntent) []domain.ActionTemplate {
	var actions []domain.ActionTemplate

	var washDevice string
	for _, app := range intent.Appliances {
		if app == "shower_1" || app == "electric_shower_1" || app == "bath_1" {
			washDevice = app
			break
		}
	}

	// Suppress the desire to clean oneself in the middle of the night
	nightSuppression := domain.UtilityBonusCurve{
		ContextKey: "time.hour_float",
		Peak:       3.0,
		Width:      1.5,
		Magnitude:  -60.0,
	}

	if intent.Preferences.WashRoutine == "quick_shower" {
		actions = append(actions, domain.ActionTemplate{
			ActionID:      "take_quick_shower",
			DeviceID:      washDevice,
			Interruptible: true,
			Satisfies: map[string]domain.ActionFill{
				"hygiene": {Amount: 80.0, Curve: "ease_out"},
			},
			BonusCurves: []domain.UtilityBonusCurve{nightSuppression}, // ATTACHED
			Duration: domain.ProbabilityDistribution{
				Type:   domain.DistributionTypeNormal,
				Mean:   "10m",
				StdDev: "2m",
			},
		})
	}

	return actions
}
