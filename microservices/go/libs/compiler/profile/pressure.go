// lib/profilecompiler/pressure.go
package profile

import (
	"github.com/tinywideclouds/go-sim-schema/domain"
)

const (
	BaseDutyDecayPerHour = 4.166
	DailyDutyDecay       = 24.0 * BaseDutyDecayPerHour
	BaseLeisureDecay     = 15.0
)

func GeneratePressure(intent PersonaIntent) ([]domain.MeterTemplate, []domain.ActionTemplate) {
	meters := []domain.MeterTemplate{
		{
			MeterID:          "work_duty",
			Max:              100.0,
			BaseDecayPerHour: BaseDutyDecayPerHour,
			Curve:            "linear",
		},
		{
			MeterID:          "leisure",
			Max:              100.0,
			BaseDecayPerHour: BaseLeisureDecay,
			Curve:            "exponential",
		},
	}

	peakMultiplier := 50.0
	for _, trait := range intent.Traits {
		if trait == "anxious" {
			peakMultiplier += 25.0
		}
		if trait == "lazy" {
			peakMultiplier -= 20.0
		}
	}

	// 1. THE COMMUTE
	// Pulls them out of the house ~45 mins before work starts.
	commuteAction := domain.ActionTemplate{
		ActionID:           "leave_house",
		Interruptible:      false,
		InitiationFriction: 20.0,
		Duration: domain.ProbabilityDistribution{
			Type:  domain.DistributionTypeConstant,
			Value: "45m",
		},
		BonusCurves: []domain.BonusCurve{
			{
				ContextKey: "time.hour_float",
				Peak:       intent.Schedule.WorkStart - 0.75, // e.g., 07:45 AM
				Width:      0.5,
				Magnitude:  peakMultiplier,
			},
		},
	}

	// 2. THE WORK SHIFT
	// Fixed duration. Fulfills the duty and handles the lunch/watercooler fakes.
	workAction := domain.ActionTemplate{
		ActionID:           "go_to_work",
		Interruptible:      false,
		InitiationFriction: 5.0, // Low friction to start once they are already there
		Satisfies: map[string]domain.ActionFill{
			"work_duty": {Amount: DailyDutyDecay, Curve: "linear"},
			"hunger":    {Amount: 30.0, Curve: "linear"},
			"leisure":   {Amount: 20.0, Curve: "linear"},
		},
		Duration: domain.ProbabilityDistribution{
			Type:  domain.DistributionTypeConstant,
			Value: "8h", // The actual fixed working hours
		},
		BonusCurves: []domain.BonusCurve{
			{
				ContextKey: "time.hour_float",
				Peak:       intent.Schedule.WorkStart + 0.25, // e.g., 08:45 AM
				Width:      0.3,                              // Sharp spike to force the shift to start
				Magnitude:  peakMultiplier * 1.5,             // Massive (WARNING AVOID 'Massive' anything - implies forced curve overrides) priority over idling
			},
		},
	}

	leisureAction := domain.ActionTemplate{
		ActionID:           "relax_on_sofa",
		Interruptible:      true,
		InitiationFriction: 0.0,
		Satisfies: map[string]domain.ActionFill{
			"leisure": {Amount: 40.0, Curve: "ease_out"},
		},
		Duration: domain.ProbabilityDistribution{
			Type:  domain.DistributionTypeConstant,
			Value: "15m",
		},
	}

	return meters, []domain.ActionTemplate{commuteAction, workAction, leisureAction}
}
