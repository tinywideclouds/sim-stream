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

type UtilityBrain interface {
	ResetMeters(actorID string, startingMeters map[string]float64)
	ApplyModifiersToMeters(actorID string, modifiers map[string]float64, limits map[string]float64)
	Process(state *engine.SimulationState, snapshot parsers.StateSnapshot, tickDuration time.Duration) ([]string, []string, []string)
	HasMeters(actorID string, costs map[string]float64) bool

	GetInterruptAction(actorID string, state *engine.SimulationState) string
	GetActionUrgency(actorID string, actionID string, state *engine.SimulationState) float64
	GetActorSnapshot(actorID string) parsers.StateSnapshot

	// ForceTask injects a continuous macro-phase (like sleep) into the biological solver
	ForceTask(actorID string, actionID string, duration time.Duration, state *engine.SimulationState)
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
	pendingReEntry map[string]*domain.AwayProfile
}

func NewStableEngine(utility UtilityBrain, routine RoutineBrain, cal CalendarProvider, samp *generator.Sampler) *StableEngine {
	return &StableEngine{
		utilityBrain:   utility,
		routineBrain:   routine,
		calendar:       cal,
		sampler:        samp,
		Burnout:        make(map[string]float64),
		activePhase:    make(map[string]string),
		pendingReEntry: make(map[string]*domain.AwayProfile),
	}
}

func (se *StableEngine) getEnergyUrgency(actorID string, state *engine.SimulationState) float64 {
	snap := se.utilityBrain.GetActorSnapshot(actorID)
	if snap == nil {
		return 0.5 // Neutral default
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

	// Strictly clamp between 0.0 and 1.0 to prevent spring explosions
	if urgency < 0.0 {
		urgency = 0.0
	}
	if urgency > 1.0 {
		urgency = 1.0
	}

	return urgency
}

// calculatePhaseTimes applies Asymmetric Spring Physics to determine the exact dynamic start and end times.
func (se *StableEngine) calculatePhaseTimes(actor domain.ActorTemplate, phase domain.DailyPhase, state *engine.SimulationState, dayType string) (time.Time, time.Time) {
	anchor, err := time.Parse("15:04", phase.AnchorTime)
	if err != nil {
		return state.SimTime, state.SimTime
	}

	baseStart := time.Date(state.SimTime.Year(), state.SimTime.Month(), state.SimTime.Day(), anchor.Hour(), anchor.Minute(), 0, 0, state.SimTime.Location())

	if state.SimTime.Hour() < 12 && anchor.Hour() > 12 {
		baseStart = baseStart.Add(-24 * time.Hour)
	} else if state.SimTime.Hour() > 12 && anchor.Hour() < 12 {
		baseStart = baseStart.Add(24 * time.Hour)
	}

	var baseDuration time.Duration
	if phase.Type == domain.PhaseTypeSleep {
		baseDuration, _ = time.ParseDuration(phase.BufferDuration)
		if baseDuration == 0 {
			baseDuration = 8 * time.Hour
		}
	} else if phase.Type == domain.PhaseTypeAway && phase.AwayProfile != nil {
		baseDuration, _ = se.sampler.Duration(phase.AwayProfile.Duration)
		if baseDuration == 0 {
			baseDuration = 8 * time.Hour
		}
	}

	baseEnd := baseStart.Add(baseDuration)
	startShift := time.Duration(0)
	endShift := time.Duration(0)

	if phase.Type == domain.PhaseTypeSleep {
		urgency := se.getEnergyUrgency(actor.ActorID, state)

		// 1. Bedtime Spring (Loose & Biological)
		if urgency > 0.5 {
			factor := (urgency - 0.5) / 0.5
			startShift = time.Duration(-45.0 * float64(time.Minute) * factor) // Max early: -45m
		} else {
			factor := (0.5 - urgency) / 0.5
			startShift = time.Duration(120.0 * float64(time.Minute) * factor) // Max late: +2h
		}

		// 2. Wakeup Spring (Stiff & Societal)
		hasWorkdayWall := false
		for _, p := range actor.Phases {
			if p.Type == domain.PhaseTypeAway {
				hasWorkdayWall = true
				break
			}
		}

		if dayType == "workday" && hasWorkdayWall {
			// Stiff Spring (Societal Pressure)
			if urgency > 0.5 {
				factor := (urgency - 0.5) / 0.5
				endShift = time.Duration(20.0 * float64(time.Minute) * factor) // Max snooze: +20m
			} else {
				factor := (0.5 - urgency) / 0.5
				endShift = time.Duration(-15.0 * float64(time.Minute) * factor) // Max early wake: -15m
			}
		} else {
			// Slack Spring (Weekend / Retired)
			if urgency > 0.5 {
				factor := (urgency - 0.5) / 0.5
				endShift = time.Duration(150.0 * float64(time.Minute) * factor) // Max sleep-in: +2.5h
			} else {
				factor := (0.5 - urgency) / 0.5
				endShift = time.Duration(-60.0 * float64(time.Minute) * factor) // Max early wake: -1h
			}
		}
	} else if phase.Type == domain.PhaseTypeAway {
		// Basic 10% elasticity for generic away phases
		urgency := se.getEnergyUrgency(actor.ActorID, state)
		maxShift := time.Duration(float64(baseDuration) * 0.10)
		pullFactor := (0.5 - urgency) * 2.0
		startShift = time.Duration(float64(maxShift) * pullFactor)
		endShift = startShift // The whole block shifts
	}

	return baseStart.Add(startShift), baseEnd.Add(endShift)
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
				if profile, exists := se.pendingReEntry[actor.ActorID]; exists && profile != nil {
					limits := make(map[string]float64)
					for _, m := range state.Blueprint.Meters {
						if m.Max > 0 {
							limits[m.MeterID] = m.Max
						} else {
							limits[m.MeterID] = 100.0
						}
					}
					se.utilityBrain.ApplyModifiersToMeters(actor.ActorID, profile.Modifiers, limits)
				}
			}

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

				actualStart, actualEnd := se.calculatePhaseTimes(*actor, phase, state, dayType)
				actualDuration := actualEnd.Sub(actualStart)
				timeSinceTrigger := state.SimTime.Sub(actualStart)

				if !state.SimTime.Before(actualStart) && timeSinceTrigger < actualDuration {

					// Only log exactly when the transition triggers
					if timeSinceTrigger < tickDuration*2 {
						slog.Info("Phase Applied (Asymmetric Elasticity)", "actor", actor.ActorID, "phase", phase.PhaseID, "shifted_start", actualStart.Format("15:04"), "shifted_wakeup", actualEnd.Format("15:04"))
					}

					remainingDuration := actualDuration - timeSinceTrigger
					if remainingDuration < 0 {
						remainingDuration = 0
					}

					if phase.Type == domain.PhaseTypeAway && dayType == "workday" {
						ledger.CurrentState = domain.ActorStateAway
						se.activePhase[actor.ActorID] = phase.PhaseID
						se.pendingReEntry[actor.ActorID] = phase.AwayProfile
						ledger.StateEndsAt = state.SimTime.Add(remainingDuration)

						// Optional: Pass an action ID if you want away time to be tracked continually
						se.utilityBrain.ForceTask(actor.ActorID, phase.PhaseID, remainingDuration, state)

						onMacroRail = true
						break

					} else if phase.Type == domain.PhaseTypeSleep {
						ledger.CurrentState = domain.ActorStateAsleep
						se.activePhase[actor.ActorID] = phase.PhaseID
						ledger.StateEndsAt = state.SimTime.Add(remainingDuration)

						// Forces the specific sleep task into the utility engine so energy recovers
						// continuously and it logs beautifully to the CSV.
						se.utilityBrain.ForceTask(actor.ActorID, phase.PhaseID, remainingDuration, state)

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
