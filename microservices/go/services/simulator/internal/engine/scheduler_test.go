package engine_test

import (
	"testing"
	"time"

	"github.com/tinywideclouds/go-power-simulator/internal/engine"

	"github.com/tinywideclouds/go-sim-probability/pkg/generator"
	"github.com/tinywideclouds/go-sim-probability/pkg/parsers"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

func TestScheduler_ScheduleDay(t *testing.T) {
	// Seed 1 gives predictable outputs for the normal distribution
	var seed [32]byte
	seed[0] = 1

	sampler := generator.NewSampler(seed)
	scheduler := engine.NewScheduler(sampler)

	// Mock an actor with a fuzzy morning routine
	actor := domain.ActorTemplate{
		ActorID: "parent_1",
		Routines: []domain.ActorRoutine{
			{
				RoutineID: "morning_prep",
				Trigger: domain.ProbabilityDistribution{
					Type:   domain.DistributionTypeNormal,
					Mean:   "07h00m",
					StdDev: "15m",
				},
				Deadline: domain.ProbabilityDistribution{
					Type:   domain.DistributionTypeNormal,
					Mean:   "08h00m",
					StdDev: "5m",
				},
			},
		},
	}

	// Base date is midnight on Jan 1, 2026
	baseDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	snap := parsers.StateSnapshot{}

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

	// Because we use a seeded RNG, we can check if it properly added the hours/minutes to the base date.
	// We expect the time to be roughly around 07:00 AM and 08:00 AM.
	startHour := sched.TargetStart.Hour()
	if startHour < 6 || startHour > 8 {
		t.Errorf("Expected target start around 7 AM, got %v", sched.TargetStart)
	}

	deadlineHour := sched.TargetDeadline.Hour()
	if deadlineHour < 7 || deadlineHour > 9 {
		t.Errorf("Expected target deadline around 8 AM, got %v", sched.TargetDeadline)
	}

	// Ensure dates align with the baseDate
	if sched.TargetStart.Year() != 2026 {
		t.Errorf("Expected year 2026, got %d", sched.TargetStart.Year())
	}
}
