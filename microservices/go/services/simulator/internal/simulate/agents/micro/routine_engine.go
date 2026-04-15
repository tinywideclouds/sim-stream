package micro

import (
	"time"

	"github.com/tinywideclouds/go-maths/pkg/probability"
	"github.com/tinywideclouds/go-power-simulator/internal/simulate/core"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

type RoutineEngine struct {
	scheduler    *Scheduler
	executor     *Executor
	dailyPlan    map[string]map[string]ScheduledRoutine
	currentDay   time.Time
	rolloverHour int
}

func NewRoutineEngine(s *probability.DistributionSampler, rolloverHour int) *RoutineEngine {
	scheduler := NewScheduler(s)
	executor := NewExecutor(s)

	return &RoutineEngine{
		scheduler:    scheduler,
		executor:     executor,
		rolloverHour: rolloverHour,
		dailyPlan:    make(map[string]map[string]ScheduledRoutine),
	}
}

func (re *RoutineEngine) GetActiveRoutineAction(actorID string, simTime time.Time, snap core.StateSnapshot) (string, string, bool) {
	actorPlan, exists := re.dailyPlan[actorID]
	if !exists {
		return "", "", false
	}

	for rID, sched := range actorPlan {
		actualStart := sched.TargetStart

		// Apply pure go-maths condition shifts
		for _, mod := range sched.Modifiers {
			if core.EvaluateCondition(mod.Condition, snap) {
				actualStart = actualStart.Add(time.Duration(mod.CompiledTransform.FlatShift))
			}
		}

		if simTime.After(actualStart) && simTime.Before(sched.TargetDeadline) {
			return rID, rID, true
		}
	}
	return "", "", false
}

func (re *RoutineEngine) Process(state *core.SimulationState, snap core.StateSnapshot, tickDuration time.Duration) ([]core.ActorTickState, []string, []string) {
	var activeHumanActors []core.ActorTickState

	midnight := time.Date(state.SimTime.Year(), state.SimTime.Month(), state.SimTime.Day(), 0, 0, 0, 0, state.SimTime.Location())
	if re.currentDay != midnight {
		re.currentDay = midnight
		re.buildDailyPlan(state, midnight, snap)
	}

	for _, a := range state.Blueprint.Actors {
		if a.AIModel != "" && a.AIModel != "routine" {
			continue
		}
		actorLedger := state.Actors[a.ActorID]

		// MACRO-STATE Check: Only process routines if they are Home
		if actorLedger.CurrentPhase != domain.PhaseTypeHome {
			continue
		}

		if actorLedger.CurrentState == domain.ActorStateRoutineActive && actorLedger.CurrentRoutineID != "" {
			var tpl *domain.RoutineTemplate
			for _, t := range state.Blueprint.RoutineTemplates {
				if t.RoutineID == actorLedger.CurrentRoutineID {
					tpl = &t
					break
				}
			}
			if tpl != nil {
				sched := re.dailyPlan[a.ActorID][actorLedger.CurrentRoutineID]
				re.executor.AdvanceRoutine(state, a.ActorID, tpl, sched.TargetDeadline)
				taskName := tpl.Tasks[actorLedger.RoutineStepIndex]
				activeHumanActors = append(activeHumanActors, core.ActorTickState{ActorID: a.ActorID, ActionID: taskName})
			}
		}
	}
	return activeHumanActors, nil, nil
}

func (re *RoutineEngine) buildDailyPlan(state *core.SimulationState, midnight time.Time, snap core.StateSnapshot) {
	for _, a := range state.Blueprint.Actors {
		if a.AIModel != "" && a.AIModel != "routine" {
			continue
		}
		plans, _ := re.scheduler.ScheduleDay(a, midnight, snap)
		re.dailyPlan[a.ActorID] = make(map[string]ScheduledRoutine)
		for _, p := range plans {
			re.dailyPlan[a.ActorID][p.RoutineID] = p
		}
	}
}
