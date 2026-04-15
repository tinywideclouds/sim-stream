package micro_test

import (
	"testing"
	"time"

	"github.com/tinywideclouds/go-maths/pkg/probability"
	"github.com/tinywideclouds/go-power-simulator/internal/simulate/agents/micro"
	"github.com/tinywideclouds/go-power-simulator/internal/simulate/core"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

func TestScheduler_ScheduleDay(t *testing.T) {
	baseSampler := probability.NewSampler([32]byte{1})
	distSampler := probability.NewDistributionSampler(baseSampler)
	scheduler := micro.NewScheduler(distSampler)

	actor := domain.Actor{
		ActorID: "parent_1",
		Routines: []domain.ActorRoutine{
			{
				RoutineID: "morning_prep",
				Trigger: domain.DynamicDistribution{
					Base: probability.SampleSpace{
						Type:              probability.NormalDistribution,
						Mean:              float64(7 * time.Hour),
						StandardDeviation: float64(15 * time.Minute),
					},
				},
				Deadline: domain.DynamicDistribution{
					Base: probability.SampleSpace{
						Type:              probability.NormalDistribution,
						Mean:              float64(8 * time.Hour),
						StandardDeviation: float64(5 * time.Minute),
					},
				},
			},
		},
	}

	baseDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	snap := core.StateSnapshot{}

	schedules, err := scheduler.ScheduleDay(actor, baseDate, snap)
	if err != nil {
		t.Fatalf("ScheduleDay failed: %v", err)
	}

	if len(schedules) != 1 {
		t.Fatalf("Expected 1 scheduled routine, got %d", len(schedules))
	}

	sched := schedules[0]
	if sched.RoutineID != "morning_prep" {
		t.Errorf("Expected routine ID 'morning_prep', got '%s'", sched.RoutineID)
	}

	startHour := sched.TargetStart.Hour()
	if startHour < 6 || startHour > 8 {
		t.Errorf("Expected target start around 7 AM, got %v", sched.TargetStart)
	}
}
