// lib/profilecompiler/compiler.go
package profile

import (
	"github.com/tinywideclouds/go-sim-schema/domain"
)

func Compile(intent PersonaIntent, icStrategy string) domain.NodeArchetype {

	tuning := GenerateTuning(intent.Traits)
	meters, sleepAction := GenerateBiology(intent)
	dutyMeters, commitmentActions := GeneratePressure(intent)
	meters = append(meters, dutyMeters...)

	hygieneActions := GenerateHygiene(intent)
	foodActions := GenerateFood(intent)

	// NEW: Generate physical hardware
	hardware := GenerateDevices(intent)

	startState := GenerateInitialConditions(intent, icStrategy)

	actions := []domain.ActionTemplate{sleepAction}
	actions = append(actions, commitmentActions...)
	actions = append(actions, hygieneActions...)
	actions = append(actions, foodActions...)

	actor := domain.ActorTemplate{
		ActorID:            intent.Archetype,
		Type:               "adult",
		AIModel:            "utility",
		StartingMeters:     startState,
		SoftmaxTemperature: tuning.SoftmaxTemperature,
	}

	node := domain.NodeArchetype{
		ArchetypeID:         intent.HouseholdID,
		Description:         "Compiled V3 Topography - Strategy: " + icStrategy,
		BaseTempC:           18.0,
		InsulationDecayRate: 0.1,
		Actors:              []domain.ActorTemplate{actor},
		Meters:              meters,
		Actions:             actions,
		Devices:             hardware, // ATTACHED HERE
	}

	return node
}
