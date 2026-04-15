package macro

import (
	"sort"
	"time"

	"github.com/tinywideclouds/go-maths/pkg/probability"
	"github.com/tinywideclouds/go-power-simulator/internal/simulate/core"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

type UtilityBrain interface {
	ResetMeters(actorID string, startingMeters map[string]float64)
	ApplyModifiersToMeters(actorID string, modifiers map[string]domain.MeterEffect, limits map[string]float64)
	Process(state *core.SimulationState, snapshot core.StateSnapshot, tickDuration time.Duration) ([]core.ActorTickState, []string, []string)
	HasMeters(actorID string, costs map[string]probability.SampleSpace) bool

	GetInterruptAction(actorID string, state *core.SimulationState) string
	GetActionUrgency(actorID string, actionID string, state *core.SimulationState) float64
	GetActorSnapshot(actorID string) core.StateSnapshot

	InterruptCurrentTask(actorID string, state *core.SimulationState) bool
	ForceTask(actorID string, taskName string, duration time.Duration, startTime time.Time, satisfies map[string]domain.ActionFill)
}

type DailyPhase struct {
	PhaseID   string
	PhaseType domain.PhaseType
	StartTime time.Time
	EndTime   time.Time
	Modifiers map[string]domain.MeterEffect
}

type StableEngine struct {
	utilityBrain   UtilityBrain
	calendar       CalendarProvider
	sampler        *probability.DistributionSampler
	dailySchedules map[string][]DailyPhase
	currentDay     time.Time
	rolloverHour   int
}

// NewStableEngine is now completely clean and free of phantom mock dependencies.
func NewStableEngine(u UtilityBrain, c CalendarProvider, s *probability.DistributionSampler) *StableEngine {
	return &StableEngine{
		utilityBrain:   u,
		calendar:       c,
		sampler:        s,
		rolloverHour:   3,
		dailySchedules: make(map[string][]DailyPhase),
	}
}

func (se *StableEngine) CalculatePhaseTimes(actorID string, state *core.SimulationState, snapshot core.StateSnapshot, dayType string) {
	var template *domain.Actor
	for _, a := range state.Blueprint.Actors {
		if a.ActorID == actorID {
			template = &a
			break
		}
	}
	if template == nil || len(template.Phases) == 0 {
		return
	}

	var validPhases []domain.Phase
	for _, p := range template.Phases {
		validPhases = append(validPhases, p)
	}

	sort.Slice(validPhases, func(i, j int) bool {
		return validPhases[i].Gravity > validPhases[j].Gravity
	})

	var scheduledPhases []DailyPhase
	baseDate := time.Date(state.SimTime.Year(), state.SimTime.Month(), state.SimTime.Day(), 0, 0, 0, 0, state.SimTime.Location())

	for _, p := range validPhases {
		anchorTime, _ := time.Parse("15:04", p.AnchorTime)
		startTime := baseDate.Add(time.Duration(anchorTime.Hour())*time.Hour + time.Duration(anchorTime.Minute())*time.Minute)

		if anchorTime.Hour() < se.rolloverHour && state.SimTime.Hour() >= se.rolloverHour {
			startTime = startTime.Add(24 * time.Hour)
		}

		var totalDuration time.Duration
		flattenedModifiers := make(map[string]domain.MeterEffect)

		for _, block := range p.Blocks {
			rollSpace := probability.SampleSpace{Type: probability.UniformDistribution, Min: 0.0, Max: 1.0}
			if se.sampler.Sample(rollSpace) > block.Probability {
				continue
			}

			blockDuration := se.sampler.SampleDuration(block.Duration.Base)

			for _, mod := range block.Duration.Modifiers {
				if core.EvaluateCondition(mod.Condition, snapshot) {
					blockDuration += time.Duration(mod.CompiledTransform.FlatShift)
				}
			}

			totalDuration += blockDuration

			for meterID, effect := range block.Modifiers {
				flattenedModifiers[meterID] = effect
			}
		}

		endTime := startTime.Add(totalDuration)

		scheduledPhases = append(scheduledPhases, DailyPhase{
			PhaseID:   p.PhaseID,
			PhaseType: p.Type,
			StartTime: startTime,
			EndTime:   endTime,
			Modifiers: flattenedModifiers,
		})
	}

	sort.Slice(scheduledPhases, func(i, j int) bool {
		return scheduledPhases[i].StartTime.Before(scheduledPhases[j].StartTime)
	})

	se.dailySchedules[actorID] = scheduledPhases
}

func (se *StableEngine) Process(state *core.SimulationState, snapshot core.StateSnapshot, tickDuration time.Duration) ([]core.ActorTickState, []string, []string) {
	rolloverTime := time.Date(state.SimTime.Year(), state.SimTime.Month(), state.SimTime.Day(), se.rolloverHour, 0, 0, 0, state.SimTime.Location())
	if state.SimTime.Before(rolloverTime) {
		rolloverTime = rolloverTime.Add(-24 * time.Hour)
	}

	if se.currentDay != rolloverTime {
		se.currentDay = rolloverTime
		dayType := se.calendar.GetDayType(state.SimTime)
		for _, a := range state.Blueprint.Actors {
			if a.AIModel == "stable" {
				se.CalculatePhaseTimes(a.ActorID, state, snapshot, dayType)
			}
		}
	}

	activeActors, anomalies, debugLogs := se.utilityBrain.Process(state, snapshot, tickDuration)

	for actorID, schedule := range se.dailySchedules {
		ledger := state.Actors[actorID]

		for _, phase := range schedule {
			if state.SimTime.After(phase.StartTime) && state.SimTime.Before(phase.EndTime) {
				ledger.CurrentPhase = phase.PhaseType

				if phase.PhaseType == domain.PhaseTypeAway || phase.PhaseType == domain.PhaseTypeSleep {
					isBusy := !se.utilityBrain.InterruptCurrentTask(actorID, state)
					if !isBusy {
						ledger.CurrentState = domain.ActorStateAsleep
						if phase.PhaseType == domain.PhaseTypeAway {
							ledger.CurrentState = domain.ActorStateRoutineActive
						}

						se.utilityBrain.ApplyModifiersToMeters(actorID, phase.Modifiers, map[string]float64{"energy": 100, "hunger": 100})
						ledger.StateEndsAt = phase.EndTime
						activeActors = append(activeActors, core.ActorTickState{ActorID: actorID, ActionID: phase.PhaseID})
					}
				}
				break
			} else if ledger.CurrentState == domain.ActorStateRoutineActive && ledger.StateEndsAt.Before(state.SimTime) {
				ledger.CurrentPhase = domain.PhaseTypeHome
				ledger.CurrentState = domain.ActorStateHomeFree
			}
		}
	}

	return activeActors, anomalies, debugLogs
}
