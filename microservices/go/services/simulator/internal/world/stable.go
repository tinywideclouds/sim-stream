package world

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/tinywideclouds/go-power-simulator/internal/engine"
	"github.com/tinywideclouds/go-sim-probability/pkg/generator"
	"github.com/tinywideclouds/go-sim-probability/pkg/parsers"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

const hourMinute = "15:04"

type UtilityBrain interface {
	ResetMeters(actorID string, startingMeters map[string]float64)
	ApplyModifiersToMeters(actorID string, modifiers map[string]domain.ContinuousEffect, limits map[string]float64)
	Process(state *engine.SimulationState, snapshot parsers.StateSnapshot, tickDuration time.Duration) ([]string, []string, []string)
	HasMeters(actorID string, costs map[string]float64) bool

	GetInterruptAction(actorID string, state *engine.SimulationState) string
	GetActionUrgency(actorID string, actionID string, state *engine.SimulationState) float64
	GetActorSnapshot(actorID string) parsers.StateSnapshot

	ForceTask(actorID string, taskName string, duration time.Duration, startTime time.Time, satisfies map[string]domain.ActionFill)
}

type RoutineBrain interface {
	GetActiveRoutineAction(actorID string, simTime time.Time, snap parsers.StateSnapshot) (string, string, bool)
	ProcessActor(actorID string, state *engine.SimulationState, tickDuration time.Duration, snap parsers.StateSnapshot) string
	AbortRoutine(actorID string)
}

type StableEngine struct {
	utilityBrain UtilityBrain
	routineBrain RoutineBrain
	calendar     CalendarProvider
	sampler      *generator.Sampler

	Burnout        map[string]float64
	activePhase    map[string]string
	pendingReEntry map[string]domain.PhaseModifiers
}

func NewStableEngine(utility UtilityBrain, routine RoutineBrain, cal CalendarProvider, samp *generator.Sampler) *StableEngine {
	return &StableEngine{
		utilityBrain:   utility,
		routineBrain:   routine,
		calendar:       cal,
		sampler:        samp,
		Burnout:        make(map[string]float64),
		activePhase:    make(map[string]string),
		pendingReEntry: make(map[string]domain.PhaseModifiers),
	}
}

func (se *StableEngine) getEnergyUrgency(actorID string, state *engine.SimulationState) float64 {
	snap := se.utilityBrain.GetActorSnapshot(actorID)
	if snap == nil {
		return 0.5
	}

	energyVal, ok := snap["actor.energy"].(float64)
	if !ok {
		return 0.5
	}

	maxEnergy := 100.0
	for _, m := range state.Blueprint.Meters {
		if m.MeterID == "energy" && m.Max > 0 {
			maxEnergy = m.Max
			break
		}
	}

	urgency := (maxEnergy - energyVal) / maxEnergy
	if urgency < 0.0 {
		urgency = 0.0
	}
	if urgency > 1.0 {
		urgency = 1.0
	}

	return urgency
}

func (se *StableEngine) CalculatePhaseTimes(actor domain.Actor, phase domain.Phase, state *engine.SimulationState, dayType string) (time.Time, time.Time, time.Duration, time.Duration) {
	if phase.Type == "sleep" {
		return se.calculateSleepPhase(actor, phase, state, dayType)
	}
	return se.calculateGenericPhase(actor, phase, state, dayType)
}

func (se *StableEngine) calculateSleepPhase(actor domain.Actor, phase domain.Phase, state *engine.SimulationState, dayType string) (time.Time, time.Time, time.Duration, time.Duration) {
	anchor, err := time.Parse(hourMinute, phase.AnchorTime)
	if err != nil {
		return state.SimTime, state.SimTime, 0, 0
	}

	normalSleepStart := time.Date(state.SimTime.Year(), state.SimTime.Month(), state.SimTime.Day(), anchor.Hour(), anchor.Minute(), 0, 0, state.SimTime.Location())
	if state.SimTime.Hour() < 12 && anchor.Hour() > 12 {
		normalSleepStart = normalSleepStart.Add(-24 * time.Hour)
	} else if state.SimTime.Hour() > 12 && anchor.Hour() < 12 {
		normalSleepStart = normalSleepStart.Add(24 * time.Hour)
	}

	baseDuration, _ := se.sampler.Duration(phase.Duration.ProbabilityDistribution)
	if baseDuration == 0 {
		baseDuration = 8 * time.Hour
	}
	normalSleepWake := normalSleepStart.Add(baseDuration)

	maxShift := phase.Duration.Flexibility
	if maxShift <= 0 {
		maxShift = 1 * time.Hour
	}

	urgency := se.getEnergyUrgency(actor.ActorID, state)
	factor := (urgency - 0.5) / 0.5

	startShift := time.Duration(-float64(maxShift) * factor)

	var endShift time.Duration
	hasWorkdayWall := false
	for _, p := range actor.Phases {
		if p.Type == "away" {
			hasWorkdayWall = true
			break
		}
	}

	if dayType == "workday" && hasWorkdayWall {
		if startShift > 0 {
			endShift = 15 * time.Minute
			if endShift > startShift {
				endShift = startShift
			}
		} else {
			endShift = time.Duration(float64(startShift) * 0.1)
		}
	} else {
		if startShift > 0 {
			endShift = startShift
		} else {
			endShift = 0
		}
	}

	actualSleepStart := normalSleepStart.Add(startShift)
	actualSleepWake := normalSleepWake.Add(endShift)

	return actualSleepStart, actualSleepWake, startShift, endShift
}

func (se *StableEngine) calculateGenericPhase(actor domain.Actor, phase domain.Phase, state *engine.SimulationState, dayType string) (time.Time, time.Time, time.Duration, time.Duration) {
	anchor, err := time.Parse(hourMinute, phase.AnchorTime)
	if err != nil {
		return state.SimTime, state.SimTime, 0, 0
	}

	baseStart := time.Date(state.SimTime.Year(), state.SimTime.Month(), state.SimTime.Day(), anchor.Hour(), anchor.Minute(), 0, 0, state.SimTime.Location())
	if state.SimTime.Hour() < 12 && anchor.Hour() > 12 {
		baseStart = baseStart.Add(-24 * time.Hour)
	} else if state.SimTime.Hour() > 12 && anchor.Hour() < 12 {
		baseStart = baseStart.Add(24 * time.Hour)
	}

	baseDuration, _ := se.sampler.Duration(phase.Duration.ProbabilityDistribution)
	if baseDuration == 0 {
		baseDuration = 8 * time.Hour
	}

	maxShift := phase.Duration.Flexibility
	if maxShift <= 0 {
		maxShift = 1 * time.Hour
	}

	urgency := se.getEnergyUrgency(actor.ActorID, state)
	pullFactor := (0.5 - urgency) * 2.0
	startShift := time.Duration(float64(maxShift) * pullFactor)
	endShift := startShift

	return baseStart.Add(startShift), baseStart.Add(baseDuration).Add(startShift), startShift, endShift
}

func (se *StableEngine) Process(state *engine.SimulationState, snapshot parsers.StateSnapshot, tickDuration time.Duration) ([]string, []string, []string) {
	var activeActors []string
	var anomalies []string
	var debugLogs []string

	dayType := se.calendar.GetDayType(state.SimTime)
	snapshot["time.day_type"] = dayType

	for i := range state.Blueprint.Actors {
		actor := &state.Blueprint.Actors[i]
		if actor.AIModel != "stable" {
			continue
		}

		ledger := state.Actors[actor.ActorID]
		if ledger == nil {
			ledger = &engine.ActorLedger{CurrentState: domain.ActorStateHomeFree}
			state.Actors[actor.ActorID] = ledger
		} else if ledger.CurrentState == 0 {
			ledger.CurrentState = domain.ActorStateHomeFree
		}

		if _, exists := se.Burnout[actor.ActorID]; !exists {
			se.Burnout[actor.ActorID] = 0.0
		}

		if se.Burnout[actor.ActorID] >= 100.0 {
			anomalies = append(anomalies, actor.ActorID+":burnout_reset")
			se.Burnout[actor.ActorID] = 0.0
			se.utilityBrain.ResetMeters(actor.ActorID, actor.StartingMeters)
			se.routineBrain.AbortRoutine(actor.ActorID)
			ledger.CurrentState = domain.ActorStateHomeFree
			ledger.StateEndsAt = state.SimTime
			delete(se.pendingReEntry, actor.ActorID)
			continue
		}

		if ledger.CurrentState == domain.ActorStateAway || ledger.CurrentState == domain.ActorStateAsleep {
			if state.SimTime.Before(ledger.StateEndsAt) {
				continue
			}

			if ledger.CurrentState == domain.ActorStateAway {
				if mods, exists := se.pendingReEntry[actor.ActorID]; exists && mods.Application == "block_end" {
					limits := make(map[string]float64)
					for _, m := range state.Blueprint.Meters {
						if m.Max > 0 {
							limits[m.MeterID] = m.Max
						} else {
							limits[m.MeterID] = 100.0
						}
					}
					se.utilityBrain.ApplyModifiersToMeters(actor.ActorID, mods.Effects, limits)
				}
			}

			slog.Info("PHASE COMPLETED", "actor", actor.ActorID, "state", ledger.CurrentState, "return_time", state.SimTime.Format(hourMinute))

			ledger.CurrentState = domain.ActorStateHomeFree
			delete(se.pendingReEntry, actor.ActorID)
			delete(se.activePhase, actor.ActorID)
		}

		onMacroRail := false

		if ledger.CurrentState == domain.ActorStateHomeFree {
			for _, phase := range actor.Phases {

				if se.activePhase[actor.ActorID] == phase.PhaseID {
					continue
				}

				actualStart, actualEnd, startShift, endShift := se.CalculatePhaseTimes(*actor, phase, state, dayType)
				actualDuration := actualEnd.Sub(actualStart)
				timeSinceTrigger := state.SimTime.Sub(actualStart)

				if !state.SimTime.Before(actualStart) && timeSinceTrigger < actualDuration {

					remainingDuration := actualDuration - timeSinceTrigger
					if remainingDuration < 0 {
						remainingDuration = 0
					}

					if phase.Type == "away" && dayType == "workday" {
						slog.Info("AWAY ENGAGED (Linear Spring)",
							"actor", actor.ActorID,
							"phase", phase.PhaseID,
							"target_leave", actualStart.Add(-startShift).Format(hourMinute),
							"actual_leave", actualStart.Format(hourMinute),
							"leave_shift", startShift.String(),
							"target_return", actualEnd.Add(-endShift).Format(hourMinute),
							"actual_return", actualEnd.Format(hourMinute),
							"return_shift", endShift.String(),
						)

						ledger.CurrentState = domain.ActorStateAway
						se.activePhase[actor.ActorID] = phase.PhaseID
						se.pendingReEntry[actor.ActorID] = phase.Modifiers
						ledger.StateEndsAt = state.SimTime.Add(remainingDuration)

						satisfies := make(map[string]domain.ActionFill)
						if phase.Modifiers.Application == "continuous" {
							for m, eff := range phase.Modifiers.Effects {
								satisfies[m] = domain.ActionFill{Amount: eff.Amount, Curve: eff.Curve}
							}
						}
						se.utilityBrain.ForceTask(actor.ActorID, phase.PhaseID, remainingDuration, state.SimTime, satisfies)

						onMacroRail = true
						break

					} else if phase.Type == "sleep" {
						slog.Info("SLEEP ENGAGED (Asymmetric Spring)",
							"actor", actor.ActorID,
							"target_bedtime", actualStart.Add(-startShift).Format(hourMinute),
							"actual_bedtime", actualStart.Format(hourMinute),
							"start_shift", startShift.String(),
							"target_wakeup", actualEnd.Add(-endShift).Format(hourMinute),
							"actual_wakeup", actualEnd.Format(hourMinute),
							"wake_shift", endShift.String(),
						)

						ledger.CurrentState = domain.ActorStateAsleep
						se.activePhase[actor.ActorID] = phase.PhaseID
						ledger.StateEndsAt = state.SimTime.Add(remainingDuration)

						satisfies := make(map[string]domain.ActionFill)
						if phase.Modifiers.Application == "continuous" {
							for m, eff := range phase.Modifiers.Effects {
								satisfies[m] = domain.ActionFill{Amount: eff.Amount, Curve: eff.Curve}
							}
						}
						se.utilityBrain.ForceTask(actor.ActorID, phase.PhaseID, remainingDuration, state.SimTime, satisfies)

						onMacroRail = true
						break
					}
				}
			}
		} else {
			onMacroRail = true
		}

		if onMacroRail {
			continue
		}

		mergedSnapshot := make(parsers.StateSnapshot)
		for k, v := range snapshot {
			mergedSnapshot[k] = v
		}
		actorSnap := se.utilityBrain.GetActorSnapshot(actor.ActorID)
		for k, v := range actorSnap {
			mergedSnapshot[k] = v
		}

		intendedActionID, routineID, isScheduled := se.routineBrain.GetActiveRoutineAction(actor.ActorID, state.SimTime, mergedSnapshot)

		if isScheduled {
			var ce domain.CollectiveEvent
			isCollectiveEvent := false
			for _, event := range state.Blueprint.CollectiveEvents {
				if event.EventID == routineID {
					ce = event
					isCollectiveEvent = true
					break
				}
			}

			abortRoutine := false

			if isCollectiveEvent {
				for _, cond := range ce.AbortConditions {
					pass, err := parsers.CheckCondition(cond, mergedSnapshot)
					if err == nil && pass {
						abortRoutine = true
						break
					}
				}
				if !abortRoutine && ce.BaseFragility > 0 {
					rollDist := domain.ProbabilityDistribution{Type: domain.DistributionTypeUniform, Min: 0.0, Max: 1.0}
					roll, _ := se.sampler.Float64(rollDist)
					if roll < ce.BaseFragility {
						abortRoutine = true
					}
				}
			}

			if !abortRoutine {
				var costs map[string]float64
				for _, act := range state.Blueprint.Actions {
					if act.ActionID == intendedActionID {
						costs = act.Costs
						break
					}
				}
				if !se.utilityBrain.HasMeters(actor.ActorID, costs) {
					se.Burnout[actor.ActorID] += 15.0
					abortRoutine = true
					debugLogs = append(debugLogs, fmt.Sprintf("[%s] V3 Rejected Routine '%s': Insufficient meters. Burnout at %.1f", actor.ActorID, routineID, se.Burnout[actor.ActorID]))
				}
			}

			if abortRoutine {
				se.routineBrain.AbortRoutine(actor.ActorID)
				anomalies = append(anomalies, actor.ActorID+":aborted_routine:"+routineID)
			} else {
				interrupt := se.utilityBrain.GetInterruptAction(actor.ActorID, state)
				if interrupt == "" {
					taskName := se.routineBrain.ProcessActor(actor.ActorID, state, tickDuration, mergedSnapshot)
					if taskName != "" {
						activeActors = append(activeActors, actor.ActorID+":"+taskName)
					}
					continue
				}
			}
		}
	}

	utilAct, utilAnom, utilDbg := se.utilityBrain.Process(state, snapshot, tickDuration)
	activeActors = append(activeActors, utilAct...)
	anomalies = append(anomalies, utilAnom...)
	debugLogs = append(debugLogs, utilDbg...)

	return activeActors, anomalies, debugLogs
}
