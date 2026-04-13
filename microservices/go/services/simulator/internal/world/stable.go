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
	Process(state *engine.SimulationState, snapshot parsers.StateSnapshot, tickDuration time.Duration) ([]engine.ActorTickState, []string, []string)
	HasMeters(actorID string, costs map[string]float64) bool

	GetInterruptAction(actorID string, state *engine.SimulationState) string
	GetActionUrgency(actorID string, actionID string, state *engine.SimulationState) float64
	GetActorSnapshot(actorID string) parsers.StateSnapshot

	InterruptCurrentTask(actorID string, state *engine.SimulationState) bool
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
	pendingReentry map[string]domain.PhaseModifiers
}

func NewStableEngine(utility UtilityBrain, routine RoutineBrain, cal CalendarProvider, samp *generator.Sampler) *StableEngine {
	return &StableEngine{
		utilityBrain:   utility,
		routineBrain:   routine,
		calendar:       cal,
		sampler:        samp,
		Burnout:        make(map[string]float64),
		activePhase:    make(map[string]string),
		pendingReentry: make(map[string]domain.PhaseModifiers),
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
	anchor, _ := time.Parse(hourMinute, phase.AnchorTime)

	baseStart := time.Date(state.SimTime.Year(), state.SimTime.Month(), state.SimTime.Day(), anchor.Hour(), anchor.Minute(), 0, 0, state.SimTime.Location())

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

	actualStart := baseStart.Add(startShift)
	actualEnd := actualStart.Add(baseDuration)

	if state.SimTime.After(actualEnd) {
		actualStart = actualStart.Add(24 * time.Hour)
		actualEnd = actualEnd.Add(24 * time.Hour)
	} else if state.SimTime.Before(actualStart.Add(-12 * time.Hour)) {
		actualStart = actualStart.Add(-24 * time.Hour)
		actualEnd = actualEnd.Add(-24 * time.Hour)
	}

	return actualStart, actualEnd, startShift, endShift
}

func (se *StableEngine) Process(state *engine.SimulationState, snapshot parsers.StateSnapshot, tickDuration time.Duration) ([]engine.ActorTickState, []string, []string) {
	var activeActors []engine.ActorTickState
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
			delete(se.pendingReentry, actor.ActorID)
			continue
		}

		if ledger.CurrentState == domain.ActorStateAway || ledger.CurrentState == domain.ActorStateAsleep {
			if state.SimTime.Before(ledger.StateEndsAt) {
				continue
			}

			if ledger.CurrentState == domain.ActorStateAway {
				if mods, exists := se.pendingReentry[actor.ActorID]; exists && mods.Application == "block_end" {
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

			snap := se.utilityBrain.GetActorSnapshot(actor.ActorID)
			var metrics string
			for k, v := range snap {
				if val, ok := v.(float64); ok {
					metrics += fmt.Sprintf("%s:%.1f ", k[6:], val)
				}
			}

			simLogger := slog.With("sim_time", state.SimTime.Format("Mon 15:04"))
			switch ledger.CurrentState {
			case domain.ActorStateAsleep:
				simLogger.Info("ACTOR WOKE UP", "actor", actor.ActorID, "state", metrics)
			case domain.ActorStateAway:
				simLogger.Info("ACTOR RETURNED HOME", "actor", actor.ActorID, "state", metrics)
			}

			simLogger.Info("PHASE COMPLETED", "actor", actor.ActorID, "state", ledger.CurrentState)

			ledger.CurrentState = domain.ActorStateHomeFree
			delete(se.pendingReentry, actor.ActorID)
			delete(se.activePhase, actor.ActorID)
		}

		onMacroRail := false

		if ledger.CurrentState == domain.ActorStateHomeFree {
			for _, phase := range actor.Phases {
				actualStart, actualEnd, startShift, endShift := se.CalculatePhaseTimes(*actor, phase, state, dayType)

				if se.activePhase[actor.ActorID] == phase.PhaseID {
					if !state.SimTime.Before(actualEnd) {
						if mods, exists := se.pendingReentry[actor.ActorID]; exists && mods.Application == "block_end" {
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
						delete(se.activePhase, actor.ActorID)
						delete(se.pendingReentry, actor.ActorID)

						simLogger := slog.With("sim_time", state.SimTime.Format("Mon 15:04"))
						simLogger.Info("PHASE COMPLETED", "actor", actor.ActorID, "phase", phase.PhaseID)
					}
					continue
				}

				actualDuration := actualEnd.Sub(actualStart)
				timeSinceTrigger := state.SimTime.Sub(actualStart)

				if !state.SimTime.Before(actualStart) && timeSinceTrigger < actualDuration {

					remainingDuration := actualDuration - timeSinceTrigger
					if remainingDuration < 0 {
						remainingDuration = 0
					}

					if phase.Type == "away" && dayType == "workday" {
						simLogger := slog.With("sim_time", state.SimTime.Format("Mon 15:04"))
						simLogger.Info("AWAY ENGAGED", "actor", actor.ActorID, "phase", phase.PhaseID, "start", startShift, "leave", actualStart.Format(hourMinute), "end", endShift, "return", actualEnd.Format(hourMinute))

						ledger.CurrentState = domain.ActorStateAway
						se.activePhase[actor.ActorID] = phase.PhaseID
						se.pendingReentry[actor.ActorID] = phase.Modifiers
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

					} else if phase.Type == "wfh" && dayType == "workday" {
						simLogger := slog.With("sim_time", state.SimTime.Format("Mon 15:04"))
						simLogger.Info("WFH SHIFT ENGAGED", "actor", actor.ActorID, "phase", phase.PhaseID, "start", actualStart.Format(hourMinute), "end", actualEnd.Format(hourMinute))

						se.activePhase[actor.ActorID] = phase.PhaseID
						se.pendingReentry[actor.ActorID] = phase.Modifiers

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
						snap := se.utilityBrain.GetActorSnapshot(actor.ActorID)
						var metrics string
						for k, v := range snap {
							if val, ok := v.(float64); ok {
								metrics += fmt.Sprintf("%s:%.1f ", k[6:], val)
							}
						}

						simLogger := slog.With("sim_time", state.SimTime.Format("Mon 15:04"))
						simLogger.Info("SLEEP ENGAGED", "actor", actor.ActorID, "bedtime", actualStart.Format(hourMinute), "wakeup", actualEnd.Format(hourMinute), "state", metrics)

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
						activeActors = append(activeActors, engine.ActorTickState{ActorID: actor.ActorID, ActionID: taskName})
					}
					continue
				}
			}
		}
	}

	// RUN THE BIOLOGY ENGINE
	utilAct, utilAnom, utilDbg := se.utilityBrain.Process(state, snapshot, tickDuration)
	activeActors = append(activeActors, utilAct...)
	anomalies = append(anomalies, utilAnom...)
	debugLogs = append(debugLogs, utilDbg...)

	// --- WFH VETO AND RESUMPTION LOGIC ---
	currentActions := make(map[string]string)
	for _, act := range activeActors {
		currentActions[act.ActorID] = act.ActionID
	}

	simLogger := slog.With("sim_time", state.SimTime.Format("Mon 15:04"))

	for actorID, phaseID := range se.activePhase {
		var activePhaseDef domain.Phase
		var aDef domain.Actor
		var isWFH bool

		for _, a := range state.Blueprint.Actors {
			if a.ActorID == actorID {
				aDef = a
				for _, p := range a.Phases {
					if p.PhaseID == phaseID && p.Type == "wfh" {
						activePhaseDef = p
						isWFH = true
						break
					}
				}
				break
			}
		}

		if !isWFH {
			continue
		}

		ledger, exists := state.Actors[actorID]
		if !exists || ledger.CurrentState != domain.ActorStateHomeFree {
			continue
		}

		actionID, isBusy := currentActions[actorID]
		isSlacking := false

		if isBusy && actionID != phaseID {
			var chosenTemplate *domain.ActionTemplate
			for _, act := range state.Blueprint.Actions {
				if act.ActionID == actionID {
					chosenTemplate = &act
					break
				}
			}

			isNecessary := false
			if chosenTemplate != nil {
				for meterID, fill := range chosenTemplate.Satisfies {
					if (meterID == "hunger" || meterID == "hygiene" || meterID == "energy") && fill.Amount > 0 {
						isNecessary = true
						break
					}
				}
			}

			if !isNecessary {
				isSlacking = true
			}
		} else if !isBusy {
			isSlacking = true
		}

		if isSlacking {
			dayType := se.calendar.GetDayType(state.SimTime)
			_, actualEnd, _, _ := se.CalculatePhaseTimes(aDef, activePhaseDef, state, dayType)
			remainingDuration := actualEnd.Sub(state.SimTime)

			if remainingDuration > 0 {
				se.utilityBrain.InterruptCurrentTask(actorID, state)

				satisfies := make(map[string]domain.ActionFill)
				if mods, ok := se.pendingReentry[actorID]; ok && mods.Application == "continuous" {
					for m, eff := range mods.Effects {
						satisfies[m] = domain.ActionFill{Amount: eff.Amount, Curve: eff.Curve}
					}
				}

				se.utilityBrain.ForceTask(actorID, phaseID, remainingDuration, state.SimTime, satisfies)
				ledger.StateEndsAt = state.SimTime.Add(remainingDuration)

				foundInArray := false
				for i, act := range activeActors {
					if act.ActorID == actorID {
						activeActors[i].ActionID = phaseID
						foundInArray = true
						break
					}
				}
				if !foundInArray {
					activeActors = append(activeActors, engine.ActorTickState{ActorID: actorID, ActionID: phaseID})
				}

				if isBusy {
					simLogger.Info("WFH SLACKING VETOED", "actor", actorID, "attempted", actionID, "resumed", phaseID)
				} else {
					simLogger.Info("WFH RESUMED AFTER BREAK", "actor", actorID, "resumed", phaseID)
				}
			}
		}
	}

	return activeActors, anomalies, debugLogs
}

// Pass-throughs for Orchestrator multi-agent coordination
func (se *StableEngine) InterruptCurrentTask(actorID string, state *engine.SimulationState) bool {
	return se.utilityBrain.InterruptCurrentTask(actorID, state)
}

func (se *StableEngine) GetActionUrgency(actorID string, actionID string, state *engine.SimulationState) float64 {
	return se.utilityBrain.GetActionUrgency(actorID, actionID, state)
}

func (se *StableEngine) ForceTask(actorID string, taskName string, duration time.Duration, startTime time.Time, satisfies map[string]domain.ActionFill) {
	se.utilityBrain.ForceTask(actorID, taskName, duration, startTime, satisfies)
}

func (se *StableEngine) GetActorSnapshot(actorID string) parsers.StateSnapshot {
	return se.utilityBrain.GetActorSnapshot(actorID)
}
