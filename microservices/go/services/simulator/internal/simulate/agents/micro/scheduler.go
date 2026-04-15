package micro

import (
	"time"

	"github.com/tinywideclouds/go-maths/pkg/probability"
	"github.com/tinywideclouds/go-power-simulator/internal/simulate/core"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

type ScheduledRoutine struct {
	RoutineID      string
	TargetStart    time.Time
	TargetDeadline time.Time
	Modifiers      []domain.DistributionModifier
	HasStarted     bool
}

type Scheduler struct {
	sampler *probability.DistributionSampler
}

func NewScheduler(sampler *probability.DistributionSampler) *Scheduler {
	return &Scheduler{sampler: sampler}
}

func (s *Scheduler) ScheduleDay(actor domain.Actor, baseDate time.Time, snap core.StateSnapshot) ([]ScheduledRoutine, error) {
	var schedules []ScheduledRoutine

	for _, r := range actor.Routines {
		// Evaluate the BASE math space
		startOffset := s.sampler.SampleDuration(r.Trigger.Base)
		targetStart := baseDate.Add(startOffset)

		deadlineOffset := s.sampler.SampleDuration(r.Deadline.Base)
		targetDeadline := baseDate.Add(deadlineOffset)

		schedules = append(schedules, ScheduledRoutine{
			RoutineID:      r.RoutineID,
			TargetStart:    targetStart,
			TargetDeadline: targetDeadline,
			Modifiers:      r.Trigger.Modifiers,
			HasStarted:     false,
		})
	}

	return schedules, nil
}
