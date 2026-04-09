package aiengine

import (
	"fmt"
	"math"
	"time"

	"github.com/tinywideclouds/go-power-simulator/internal/engine"
	"github.com/tinywideclouds/go-sim-probability/pkg/generator"
	"github.com/tinywideclouds/go-sim-probability/pkg/parsers"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

func evaluateGaussianBonus(curve domain.BonusCurve, currentValue float64) float64 {
	distance := math.Abs(currentValue - curve.Peak)
	if curve.ContextKey == "time.hour_float" && distance > 12.0 {
		distance = 24.0 - distance
	}
	exponent := -math.Pow(distance, 2) / (2 * math.Pow(curve.Width, 2))
	return curve.Magnitude * math.Exp(exponent)
}

func (ue *UtilityEngine) GetInterruptAction(actorID string, state *engine.SimulationState) string {
	actorMeters, exists := ue.meters[actorID]
	if !exists {
		return ""
	}

	criticalThreshold := 15.0
	var mostCriticalMeter string
	var lowestValue float64 = 100.0

	for meterID, value := range actorMeters {
		if value < criticalThreshold && value < lowestValue {
			lowestValue = value
			mostCriticalMeter = meterID
		}
	}

	if mostCriticalMeter == "" {
		return ""
	}

	var bestAction string
	var highestImpact float64 = 0.0
	var actorType string
	for _, a := range state.Blueprint.Actors {
		if a.ActorID == actorID {
			actorType = a.Type
			break
		}
	}

	for _, action := range state.Blueprint.Actions {
		hasPermission := false
		if len(action.ActorTags) == 0 {
			hasPermission = true
		} else {
			for _, tag := range action.ActorTags {
				if tag == actorType || tag == "any" {
					hasPermission = true
					break
				}
			}
		}
		if !hasPermission {
			continue
		}
		if fill, exists := action.Satisfies[mostCriticalMeter]; exists && fill.Amount > highestImpact {
			highestImpact = fill.Amount
			bestAction = action.ActionID
		}
	}

	return bestAction
}

type ActiveTask struct {
	ActionID      string
	StartTime     time.Time
	TotalDuration time.Duration
	Satisfies     map[string]domain.ActionFill
}

type UtilityEngine struct {
	sampler      *generator.Sampler
	meters       map[string]map[string]float64
	activeAction map[string]string
	activeTasks  map[string]*ActiveTask
	sharedState  map[string]float64
}

func NewUtilityEngine(sampler *generator.Sampler) *UtilityEngine {
	return &UtilityEngine{
		sampler:      sampler,
		meters:       make(map[string]map[string]float64),
		activeAction: make(map[string]string),
		activeTasks:  make(map[string]*ActiveTask),
		sharedState:  make(map[string]float64),
	}
}

func (ue *UtilityEngine) ResetMeters(actorID string, startingMeters map[string]float64) {
	if _, exists := ue.meters[actorID]; !exists {
		ue.meters[actorID] = make(map[string]float64)
	}
	for k, v := range startingMeters {
		ue.meters[actorID][k] = v
	}
}

// Updated to accept the new ContinuousEffect mapping for block_end applications
func (ue *UtilityEngine) ApplyModifiersToMeters(actorID string, modifiers map[string]domain.ContinuousEffect, limits map[string]float64) {
	if _, exists := ue.meters[actorID]; !exists {
		return
	}
	for meterID, effect := range modifiers {
		ue.meters[actorID][meterID] += effect.Amount
		if ue.meters[actorID][meterID] < 0.0 {
			ue.meters[actorID][meterID] = 0.0
		}
		if limit, exists := limits[meterID]; exists && ue.meters[actorID][meterID] > limit {
			ue.meters[actorID][meterID] = limit
		}
	}
}

func (ue *UtilityEngine) HasMeters(actorID string, costs map[string]float64) bool {
	actorMeters, exists := ue.meters[actorID]
	if !exists {
		return false
	}
	for meterID, cost := range costs {
		currentVal, ok := actorMeters[meterID]
		if ok && currentVal < cost {
			return false
		}
	}
	return true
}

func (ue *UtilityEngine) GetActorSnapshot(actorID string) parsers.StateSnapshot {
	actorMeters, exists := ue.meters[actorID]
	if !exists {
		return nil
	}
	snapshot := make(parsers.StateSnapshot)
	for meterID, value := range actorMeters {
		snapshot["actor."+meterID] = value
	}
	return snapshot
}

func (ue *UtilityEngine) GetActionUrgency(actorID string, actionID string, state *engine.SimulationState) float64 {
	actorMeters, exists := ue.meters[actorID]
	if !exists {
		return 0.0
	}

	var targetAction domain.ActionTemplate
	found := false
	for _, act := range state.Blueprint.Actions {
		if act.ActionID == actionID {
			targetAction = act
			found = true
			break
		}
	}
	if !found {
		return 0.0
	}

	maxMeters := make(map[string]float64)
	for _, m := range state.Blueprint.Meters {
		if m.Max > 0 {
			maxMeters[m.MeterID] = m.Max
		} else {
			maxMeters[m.MeterID] = 100.0
		}
	}

	urgency := 0.0
	for meterID, fill := range targetAction.Satisfies {
		currentVal, hasMeter := actorMeters[meterID]
		if !hasMeter {
			continue
		}
		maxVal := maxMeters[meterID]
		deficit := maxVal - currentVal
		if deficit > 0 {
			deficitRatio := deficit / maxVal
			urgency += deficitRatio * fill.Amount
		}
	}
	return urgency
}

// Updated to accept the translated ActionFill maps directly from the Arbiter
func (ue *UtilityEngine) ForceTask(actorID string, taskName string, duration time.Duration, startTime time.Time, satisfies map[string]domain.ActionFill) {
	delete(ue.activeTasks, actorID)
	delete(ue.activeAction, actorID)

	ue.activeAction[actorID] = taskName
	ue.activeTasks[actorID] = &ActiveTask{
		ActionID:      taskName,
		StartTime:     startTime,
		TotalDuration: duration,
		Satisfies:     satisfies,
	}
}

func calculateCurveIntegral(curve string, p float64) float64 {
	if p <= 0.0 {
		return 0.0
	}
	if p >= 1.0 {
		return 1.0
	}

	switch curve {
	case "ease_in", "back_loaded":
		return p * p
	case "ease_out", "front_loaded":
		return 1.0 - math.Pow(1.0-p, 2)
	case "bell":
		return p * p * (3.0 - 2.0*p)
	case "linear":
		fallthrough
	default:
		return p
	}
}

func (ue *UtilityEngine) Process(state *engine.SimulationState, snapshot parsers.StateSnapshot, tickDuration time.Duration) ([]string, []string, []string) {
	var activeHumanActors []string
	var anomalies []string
	var debugLogs []string

	maxMeters := make(map[string]float64)
	for _, m := range state.Blueprint.Meters {
		if m.Max > 0 {
			maxMeters[m.MeterID] = m.Max
		} else {
			maxMeters[m.MeterID] = 100.0
		}
	}

	for _, actor := range state.Blueprint.Actors {
		if actor.AIModel != "utility" && actor.AIModel != "stable" {
			continue
		}

		if _, exists := ue.meters[actor.ActorID]; !exists {
			ue.meters[actor.ActorID] = make(map[string]float64)
			for k, v := range actor.StartingMeters {
				ue.meters[actor.ActorID][k] = v
			}
		}

		actorLedger := state.Actors[actor.ActorID]

		if actorLedger.CurrentState == domain.ActorStateAway {
			continue
		}

		// 1. BASE DECAY APPLICATION
		for _, m := range state.Blueprint.Meters {
			decayAmount := m.BaseDecayPerHour * (tickDuration.Seconds() / 3600.0)
			if m.Curve == "exponential" {
				currentVal := ue.meters[actor.ActorID][m.MeterID]
				factor := (maxMeters[m.MeterID] - currentVal) / maxMeters[m.MeterID]
				if factor < 0.1 {
					factor = 0.1
				}
				decayAmount *= (1.0 + factor)
			}
			ue.meters[actor.ActorID][m.MeterID] -= decayAmount
			if ue.meters[actor.ActorID][m.MeterID] < 0 {
				ue.meters[actor.ActorID][m.MeterID] = 0
			}
		}

		// 2. ACTIVE TASK EVALUATION
		if state.SimTime.Before(actorLedger.StateEndsAt) {
			activeHumanActors = append(activeHumanActors, actor.ActorID+":"+ue.activeAction[actor.ActorID])

			if task, exists := ue.activeTasks[actor.ActorID]; exists {
				elapsedNow := state.SimTime.Sub(task.StartTime)
				elapsedPrev := elapsedNow - tickDuration

				pNow := float64(elapsedNow) / float64(task.TotalDuration)
				pPrev := float64(elapsedPrev) / float64(task.TotalDuration)

				for meterID, fill := range task.Satisfies {
					integralNow := calculateCurveIntegral(fill.Curve, pNow)
					integralPrev := calculateCurveIntegral(fill.Curve, pPrev)

					deltaPercentage := integralNow - integralPrev
					if deltaPercentage > 0 {
						ue.meters[actor.ActorID][meterID] += fill.Amount * deltaPercentage
						limit := maxMeters[meterID]
						if ue.meters[actor.ActorID][meterID] > limit {
							ue.meters[actor.ActorID][meterID] = limit
						}
					}
				}

				var mLog string
				for m, v := range ue.meters[actor.ActorID] {
					mLog += fmt.Sprintf("%s:%.1f, ", m, v)
				}
				debugLogs = append(debugLogs, fmt.Sprintf("[%s] (Busy: %s) Live Meters: %s", actor.ActorID, task.ActionID, mLog))
			}
			continue
		}

		// 3. CLEANUP & IDLE PREP
		if _, wasBusy := ue.activeAction[actor.ActorID]; wasBusy {
			delete(ue.activeAction, actor.ActorID)
			delete(ue.activeTasks, actor.ActorID)
		}

		// 4. ACTION SCORING
		bestScore := 0.0
		var validActions []domain.ActionTemplate
		actionScores := make(map[string]float64)

		for _, template := range state.Blueprint.Actions {
			available := true
			for _, cond := range template.AvailableWhen {
				if pass, _ := parsers.CheckCondition(cond, snapshot); !pass {
					available = false
					break
				}
			}
			if !available {
				continue
			}

			score := 0.0
			for meterID, fill := range template.Satisfies {
				currentVal := ue.meters[actor.ActorID][meterID]
				deficit := maxMeters[meterID] - currentVal

				fillAmount := fill.Amount
				if fillAmount > deficit {
					fillAmount = deficit
				}

				rawUrgency := deficit / maxMeters[meterID]
				curveType := "linear"
				for _, m := range state.Blueprint.Meters {
					if m.MeterID == meterID {
						curveType = m.Curve
						break
					}
				}

				urgency := applyUrgencyCurve(curveType, rawUrgency)
				score += fillAmount * urgency
			}

			for _, bonus := range template.BonusCurves {
				if val, ok := snapshot[bonus.ContextKey].(float64); ok {
					score += evaluateGaussianBonus(bonus, val)
				}
			}

			if ue.activeAction[actor.ActorID] != template.ActionID {
				score -= template.InitiationFriction
			}

			if score > 0 {
				validActions = append(validActions, template)
				actionScores[template.ActionID] = score
				if score > bestScore {
					bestScore = score
				}
			}
		}

		// 5. ROULETTE WHEEL
		var wheel []string
		var weights []float64
		var totalWeight float64

		temperature := actor.SoftmaxTemperature
		if temperature <= 0.0 {
			temperature = 1.0
		}

		for _, act := range validActions {
			score := actionScores[act.ActionID]
			if score >= (bestScore * 0.5) {
				weight := math.Exp((score - bestScore) / temperature)
				wheel = append(wheel, act.ActionID)
				weights = append(weights, weight)
				totalWeight += weight
			}
		}

		// 6. EXECUTE CHOSEN ACTION
		if len(wheel) > 0 {
			rollDist := domain.ProbabilityDistribution{Type: domain.DistributionTypeUniform, Min: 0.0, Max: 1.0}
			fVal, err := ue.sampler.Float64(rollDist)
			if err != nil {
				fVal = 0.5
			}

			pick := fVal * totalWeight
			chosenIndex := 0
			for i, w := range weights {
				pick -= w
				if pick <= 0 {
					chosenIndex = i
					break
				}
			}

			actionID := wheel[chosenIndex]
			ue.activeAction[actor.ActorID] = actionID

			var template *domain.ActionTemplate
			for _, act := range state.Blueprint.Actions {
				if act.ActionID == actionID {
					template = &act
					break
				}
			}

			if template != nil {
				duration, err := ue.sampler.Duration(template.Duration)
				if err != nil {
					duration = 15 * time.Minute
				}

				if template.DeviceID != "" {
					if devLedger, exists := state.Devices[template.DeviceID]; exists {
						devLedger.State = domain.DeviceStateOn
						devLedger.StateEndsAt = state.SimTime.Add(duration)
					}
				}

				var mLog string
				for m, v := range ue.meters[actor.ActorID] {
					mLog += fmt.Sprintf("%s:%.1f, ", m, v)
				}
				debugLogs = append(debugLogs, fmt.Sprintf("[%s] Meters(%s) | Roulette Chose: %s | Max Was: %.1f", actor.ActorID, mLog, actionID, bestScore))

				for meterID, amount := range template.Costs {
					ue.meters[actor.ActorID][meterID] -= amount
					if ue.meters[actor.ActorID][meterID] < 0 {
						ue.meters[actor.ActorID][meterID] = 0
					}
				}

				ue.activeTasks[actor.ActorID] = &ActiveTask{
					ActionID:      actionID,
					StartTime:     state.SimTime,
					TotalDuration: duration,
					Satisfies:     template.Satisfies,
				}

				actorLedger.StateEndsAt = state.SimTime.Add(duration)
				activeHumanActors = append(activeHumanActors, actor.ActorID+":"+actionID)
			}
		}
	}

	return activeHumanActors, anomalies, debugLogs
}

func applyUrgencyCurve(curve string, p float64) float64 {
	if p <= 0.0 {
		return 0.0
	}
	if p >= 1.0 {
		return 1.0
	}
	switch curve {
	case "front_loaded":
		return 1.0 - math.Pow(1.0-p, 3)
	case "exponential", "back_loaded":
		return math.Pow(p, 3)
	case "s_curve":
		return p * p * (3.0 - 2.0*p)
	case "linear":
		fallthrough
	default:
		return p
	}
}
