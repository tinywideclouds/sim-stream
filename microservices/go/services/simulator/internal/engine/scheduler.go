package engine

import (
	"time"

	"github.com/tinywideclouds/go-sim-probability/pkg/generator"
	"github.com/tinywideclouds/go-sim-probability/pkg/parsers"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

// ScheduledRoutine holds the exact, mathematically rolled times for a specific routine.
type ScheduledRoutine struct {
	RoutineID      string
	TargetStart    time.Time
	TargetDeadline time.Time
	Modifiers      []domain.DistributionModifier // NEW: Store for live evaluation
	HasStarted     bool
}

type Scheduler struct {
	sampler *generator.Sampler
}

func NewScheduler(sampler *generator.Sampler) *Scheduler {
	return &Scheduler{sampler: sampler}
}

// ScheduleDay calculates the baseline start and deadline times.
func (s *Scheduler) ScheduleDay(actor domain.ActorTemplate, baseDate time.Time, snap parsers.StateSnapshot) ([]ScheduledRoutine, error) {
	var schedules []ScheduledRoutine

	for _, r := range actor.Routines {
		// Roll baseline start
		startOffset, err := s.sampler.Duration(r.Trigger)
		if err != nil {
			return nil, err
		}
		targetStart := baseDate.Add(startOffset)

		// Roll baseline deadline
		deadlineOffset, err := s.sampler.Duration(r.Deadline)
		if err != nil {
			return nil, err
		}
		targetDeadline := baseDate.Add(deadlineOffset)

		schedules = append(schedules, ScheduledRoutine{
			RoutineID:      r.RoutineID,
			TargetStart:    targetStart,
			TargetDeadline: targetDeadline,
			Modifiers:      r.Trigger.Modifiers, // Pass rules to the live engine
			HasStarted:     false,
		})
	}

	return schedules, nil
}
