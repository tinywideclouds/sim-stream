package micro_test

import (
	"testing"
	"time"

	"github.com/tinywideclouds/go-maths/pkg/probability"
	"github.com/tinywideclouds/go-power-simulator/internal/simulate/agents/micro"
	"github.com/tinywideclouds/go-power-simulator/internal/simulate/core"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

func TestRoutineEngine_Adapters(t *testing.T) {
	baseSampler := probability.NewSampler([32]byte{})
	sampler := probability.NewDistributionSampler(baseSampler)

	// Create the RoutineEngine and explicitly use the exported API
	re := micro.NewRoutineEngine(sampler, 3)

	actorID := "test_actor"
	simTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	snap := make(core.StateSnapshot)

	// Since we hid dailyPlan (it's unexported now), we test it by triggering Process()
	// and verifying it gracefully handles actors without routines without crashing.

	state := &core.SimulationState{
		SimTime: simTime,
		Blueprint: &domain.NodeArchetype{
			Actors: []domain.Actor{
				{ActorID: actorID, AIModel: "routine"},
			},
		},
		Actors: map[string]*core.ActorLedger{
			actorID: {CurrentPhase: domain.PhaseTypeHome},
		},
	}

	activeActors, _, _ := re.Process(state, snap, 1*time.Minute)

	// Should be 0 since the test_actor has no routines defined in their array
	if len(activeActors) != 0 {
		t.Errorf("Expected 0 active actors, got %d", len(activeActors))
	}
}
