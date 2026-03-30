package engine

import (
	"time"

	"github.com/tinywideclouds/go-sim-probability/pkg/generator"
	"github.com/tinywideclouds/go-sim-probability/pkg/parsers"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

// ScheduledRoutine holds the exact, mathematically rolled times for a specific routine for the current day.
type ScheduledRoutine struct {
	RoutineID      string
	TargetStart    time.Time
	TargetDeadline time.Time
	HasStarted     bool
}

// Scheduler handles the daily fuzzy math for all actors.
type Scheduler struct {
	sampler *generator.Sampler
}

// NewScheduler creates a new daily scheduler.
func NewScheduler(sampler *generator.Sampler) *Scheduler {
	return &Scheduler{sampler: sampler}
}

// ScheduleDay calculates the concrete start and deadline times for a single actor's routines.
// It applies environmental modifiers (like weather) before rolling the dice.
func (s *Scheduler) ScheduleDay(actor domain.ActorTemplate, baseDate time.Time, ctx parsers.EnvironmentSnapshot) ([]ScheduledRoutine, error) {
	var schedules []ScheduledRoutine

	for _, r := range actor.Routines {
		// 1. Apply Modifiers to the Trigger (Start Time)
		modTrigger, err := parsers.ApplyModifiers(r.Trigger, ctx)
		if err != nil {
			return nil, err
		}

		// Roll the modified Trigger
		startOffset, err := s.sampler.Duration(modTrigger)
		if err != nil {
			return nil, err
		}
		targetStart := baseDate.Add(startOffset)

		// 2. Apply Modifiers to the Deadline (End Time)
		modDeadline, err := parsers.ApplyModifiers(r.Deadline, ctx)
		if err != nil {
			return nil, err
		}

		// Roll the modified Deadline
		deadlineOffset, err := s.sampler.Duration(modDeadline)
		if err != nil {
			return nil, err
		}
		targetDeadline := baseDate.Add(deadlineOffset)

		schedules = append(schedules, ScheduledRoutine{
			RoutineID:      r.RoutineID,
			TargetStart:    targetStart,
			TargetDeadline: targetDeadline,
		})
	}

	return schedules, nil
}
