package aiengine

import (
	"time"

	"github.com/tinywideclouds/go-power-simulator/internal/engine"
	"github.com/tinywideclouds/go-sim-probability/pkg/parsers"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

// RoutineEngine implements AIEngine using predictable, scheduled routines.
// It uses a Scheduler to plan the day, a Negotiator to resolve conflicts,
// and an Executor to step through linear task lists.
type RoutineEngine struct {
	scheduler    *engine.Scheduler
	negotiator   *engine.Negotiator
	executor     *engine.Executor
	dailyPlan    map[string]map[string]engine.ScheduledRoutine
	currentDay   time.Time
	rolloverHour int
}

// NewRoutineEngine initializes the scheduled AI model.
func NewRoutineEngine(s *engine.Scheduler, n *engine.Negotiator, e *engine.Executor, rolloverHour int) *RoutineEngine {
	return &RoutineEngine{
		scheduler:    s,
		negotiator:   n,
		executor:     e,
		rolloverHour: rolloverHour,
		dailyPlan:    make(map[string]map[string]engine.ScheduledRoutine),
	}
}

// Process satisfies the AIEngine interface.
func (re *RoutineEngine) Process(state *engine.SimulationState, snap parsers.EnvironmentSnapshot, tickDuration time.Duration) ([]string, []string, []string) {
	var activeHumanActors []string

	// 1. Rollover Check (Midnight / Start of Day logic)
	logicalDay := state.SimTime
	if state.SimTime.Hour() < re.rolloverHour {
		logicalDay = state.SimTime.Add(-24 * time.Hour)
	}
	midnightOfLogicalDay := logicalDay.Truncate(24 * time.Hour)

	if !re.currentDay.Equal(midnightOfLogicalDay) || len(re.dailyPlan) == 0 {
		re.currentDay = midnightOfLogicalDay
		re.buildDailyPlan(state, midnightOfLogicalDay, snap)
	}

	// 2. The Routine AI Loop
	for _, actorTpl := range state.Blueprint.Actors {
		// Only process actors assigned to this specific engine
		if actorTpl.AIModel != "" && actorTpl.AIModel != "routine" {
			continue
		}

		actorLedger := state.Actors[actorTpl.ActorID]
		actorPlan := re.dailyPlan[actorTpl.ActorID]

		// Wake up / Start Routine Check
		if actorLedger.CurrentState == domain.ActorStateAsleep || actorLedger.CurrentState == domain.ActorStateHomeFree {
			for routineID, routinePlan := range actorPlan {
				if !routinePlan.HasStarted && !state.SimTime.Before(routinePlan.TargetStart) && state.SimTime.Before(routinePlan.TargetDeadline) {
					routinePlan.HasStarted = true
					actorPlan[routineID] = routinePlan

					actorLedger.CurrentState = domain.ActorStateRoutineActive
					actorLedger.CurrentRoutineID = routinePlan.RoutineID
					actorLedger.RoutineStepIndex = 0
					actorLedger.StateEndsAt = state.SimTime
					break
				}
			}
		}

		// Execute Routine Step
		if actorLedger.CurrentState == domain.ActorStateRoutineActive {
			routinePlan := actorPlan[actorLedger.CurrentRoutineID]
			var tpl *domain.RoutineTemplate
			for _, rt := range state.Blueprint.RoutineTemplates {
				if rt.RoutineID == actorLedger.CurrentRoutineID {
					tpl = &rt
					break
				}
			}
			if tpl != nil {
				_ = re.executor.AdvanceRoutine(state, actorTpl.ActorID, tpl, routinePlan.TargetDeadline)
			}

			if actorLedger.CurrentState == domain.ActorStateRoutineActive {
				taskName := "transitioning"
				if tpl != nil {
					idx := actorLedger.RoutineStepIndex - 1
					if idx >= 0 && idx < len(tpl.Tasks) {
						taskName = tpl.Tasks[idx]
					}
				}
				activeHumanActors = append(activeHumanActors, actorTpl.ActorID+":"+taskName)
			}
		}
	}

	return activeHumanActors, nil, nil
}

// buildDailyPlan calls the Scheduler and Negotiator at the Rollover time.
func (re *RoutineEngine) buildDailyPlan(state *engine.SimulationState, midnight time.Time, snap parsers.EnvironmentSnapshot) {
	for _, a := range state.Blueprint.Actors {
		if a.AIModel != "" && a.AIModel != "routine" {
			continue
		}
		plans, _ := re.scheduler.ScheduleDay(a, midnight, snap)
		re.dailyPlan[a.ActorID] = make(map[string]engine.ScheduledRoutine)
		for _, p := range plans {
			re.dailyPlan[a.ActorID][p.RoutineID] = p
		}
	}

	for _, ce := range state.Blueprint.CollectiveEvents {
		leadPlan, exists := re.dailyPlan[ce.LeadActor][ce.Action]
		if !exists {
			continue
		}
		leadPart := engine.Participant{ActorID: ce.LeadActor, TargetTime: leadPlan.TargetDeadline, Weight: 1.0}

		var deps []engine.Participant
		for _, d := range ce.DependentActors {
			depPlan := re.dailyPlan[d.ActorID][ce.Action]
			patience, _ := time.ParseDuration(d.PatienceLimit)
			deps = append(deps, engine.Participant{
				ActorID: d.ActorID, TargetTime: depPlan.TargetDeadline,
				Weight: d.FrictionWeight, PatienceLimit: patience,
			})
		}

		finalDeadline := re.negotiator.ResolveEventTime(leadPart, deps)

		leadPlan.TargetDeadline = finalDeadline
		re.dailyPlan[ce.LeadActor][ce.Action] = leadPlan
		for _, d := range ce.DependentActors {
			depPlan := re.dailyPlan[d.ActorID][ce.Action]
			depPlan.TargetDeadline = finalDeadline
			re.dailyPlan[d.ActorID][ce.Action] = depPlan
		}
	}
}
