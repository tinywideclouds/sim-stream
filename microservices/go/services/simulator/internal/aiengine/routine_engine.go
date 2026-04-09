package aiengine

import (
	"time"

	"github.com/tinywideclouds/go-power-simulator/internal/engine"
	"github.com/tinywideclouds/go-sim-probability/pkg/parsers"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

type RoutineEngine struct {
	scheduler    *engine.Scheduler
	negotiator   *engine.Negotiator
	executor     *engine.Executor
	dailyPlan    map[string]map[string]engine.ScheduledRoutine
	currentDay   time.Time
	rolloverHour int
}

func NewRoutineEngine(s *engine.Scheduler, n *engine.Negotiator, e *engine.Executor, rolloverHour int) *RoutineEngine {
	return &RoutineEngine{
		scheduler:    s,
		negotiator:   n,
		executor:     e,
		rolloverHour: rolloverHour,
		dailyPlan:    make(map[string]map[string]engine.ScheduledRoutine),
	}
}

// ------------------------------------------------------------------
// ADAPTER APIs FOR THE STABLE ENGINE (ARBITER)
// ------------------------------------------------------------------

// GetActiveRoutineAction dynamically shifts the routine time using the live snapshot.
func (re *RoutineEngine) GetActiveRoutineAction(actorID string, simTime time.Time, snap parsers.StateSnapshot) (string, string, bool) {
	actorPlan, exists := re.dailyPlan[actorID]
	if !exists {
		return "", "", false
	}

	for rID, routinePlan := range actorPlan {
		// LIVE ELASTICITY: Calculate biological warp delta
		shift, _ := parsers.CalculateShiftDuration(routinePlan.Modifiers, snap)
		liveStart := routinePlan.TargetStart.Add(shift)
		liveDeadline := routinePlan.TargetDeadline.Add(shift)

		if !simTime.Before(liveStart) && simTime.Before(liveDeadline) {
			return rID, rID, true
		}
	}

	return "", "", false
}

func (re *RoutineEngine) AbortRoutine(actorID string) {
	actorPlan, exists := re.dailyPlan[actorID]
	if !exists {
		return
	}
	for rID, routinePlan := range actorPlan {
		routinePlan.TargetDeadline = time.Time{}
		actorPlan[rID] = routinePlan
	}
}

func (re *RoutineEngine) ProcessActor(actorID string, state *engine.SimulationState, tickDuration time.Duration, snap parsers.StateSnapshot) string {
	actorLedger := state.Actors[actorID]
	actorPlan := re.dailyPlan[actorID]

	if actorLedger.CurrentState == domain.ActorStateAsleep || actorLedger.CurrentState == domain.ActorStateHomeFree {
		for routineID, routinePlan := range actorPlan {
			shift, _ := parsers.CalculateShiftDuration(routinePlan.Modifiers, snap)
			liveStart := routinePlan.TargetStart.Add(shift)
			liveDeadline := routinePlan.TargetDeadline.Add(shift)

			if !routinePlan.HasStarted && !state.SimTime.Before(liveStart) && state.SimTime.Before(liveDeadline) {
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

	if actorLedger.CurrentState == domain.ActorStateRoutineActive {
		routinePlan := actorPlan[actorLedger.CurrentRoutineID]
		shift, _ := parsers.CalculateShiftDuration(routinePlan.Modifiers, snap)
		liveDeadline := routinePlan.TargetDeadline.Add(shift)

		var tpl *domain.RoutineTemplate
		for _, rt := range state.Blueprint.RoutineTemplates {
			if rt.RoutineID == actorLedger.CurrentRoutineID {
				tpl = &rt
				break
			}
		}
		if tpl != nil {
			_ = re.executor.AdvanceRoutine(state, actorID, tpl, liveDeadline)
		}

		if actorLedger.CurrentState == domain.ActorStateRoutineActive {
			taskName := "transitioning"
			if tpl != nil {
				idx := actorLedger.RoutineStepIndex - 1
				if idx >= 0 && idx < len(tpl.Tasks) {
					taskName = tpl.Tasks[idx]
				}
			}
			return taskName
		}
	}

	return ""
}

// ------------------------------------------------------------------

func (re *RoutineEngine) Process(state *engine.SimulationState, snap parsers.StateSnapshot, tickDuration time.Duration) ([]string, []string, []string) {
	var activeHumanActors []string

	logicalDay := state.SimTime
	if state.SimTime.Hour() < re.rolloverHour {
		logicalDay = state.SimTime.Add(-24 * time.Hour)
	}
	midnightOfLogicalDay := logicalDay.Truncate(24 * time.Hour)

	if !re.currentDay.Equal(midnightOfLogicalDay) || len(re.dailyPlan) == 0 {
		re.currentDay = midnightOfLogicalDay
		re.buildDailyPlan(state, midnightOfLogicalDay, snap)
	}

	for _, actorTpl := range state.Blueprint.Actors {
		if actorTpl.AIModel != "" && actorTpl.AIModel != "routine" {
			continue
		}

		actorLedger := state.Actors[actorTpl.ActorID]
		actorPlan := re.dailyPlan[actorTpl.ActorID]

		if actorLedger.CurrentState == domain.ActorStateAsleep || actorLedger.CurrentState == domain.ActorStateHomeFree {
			for routineID, routinePlan := range actorPlan {
				shift, _ := parsers.CalculateShiftDuration(routinePlan.Modifiers, snap)
				liveStart := routinePlan.TargetStart.Add(shift)
				liveDeadline := routinePlan.TargetDeadline.Add(shift)

				if !routinePlan.HasStarted && !state.SimTime.Before(liveStart) && state.SimTime.Before(liveDeadline) {
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

		if actorLedger.CurrentState == domain.ActorStateRoutineActive {
			routinePlan := actorPlan[actorLedger.CurrentRoutineID]
			shift, _ := parsers.CalculateShiftDuration(routinePlan.Modifiers, snap)
			liveDeadline := routinePlan.TargetDeadline.Add(shift)

			var tpl *domain.RoutineTemplate
			for _, rt := range state.Blueprint.RoutineTemplates {
				if rt.RoutineID == actorLedger.CurrentRoutineID {
					tpl = &rt
					break
				}
			}
			if tpl != nil {
				_ = re.executor.AdvanceRoutine(state, actorTpl.ActorID, tpl, liveDeadline)
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

func (re *RoutineEngine) buildDailyPlan(state *engine.SimulationState, midnight time.Time, snap parsers.StateSnapshot) {
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
