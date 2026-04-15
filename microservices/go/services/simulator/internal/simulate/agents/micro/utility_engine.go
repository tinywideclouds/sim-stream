package micro

import (
	"math"
	"time"

	"github.com/tinywideclouds/go-maths/pkg/geom"
	"github.com/tinywideclouds/go-maths/pkg/probability"
	"github.com/tinywideclouds/go-power-simulator/internal/simulate/core"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

type ActiveTask struct {
	ActionID      string
	StartTime     time.Time
	TotalDuration time.Duration
	Satisfies     map[string]domain.ActionFill
}

type UtilityEngine struct {
	sampler      *probability.DistributionSampler
	meters       map[string]map[string]float64
	activeAction map[string]string
	activeTasks  map[string]*ActiveTask
	sharedState  map[string]float64
}

func NewUtilityEngine(sampler *probability.DistributionSampler) *UtilityEngine {
	return &UtilityEngine{
		sampler:      sampler,
		meters:       make(map[string]map[string]float64),
		activeAction: make(map[string]string),
		activeTasks:  make(map[string]*ActiveTask),
		sharedState:  make(map[string]float64),
	}
}

func evaluateGaussianBonus(curve domain.BonusCurve, currentValue float64) float64 {
	distance := math.Abs(currentValue - curve.Peak)
	if curve.ContextKey == "time.hour_float" && distance > 12.0 {
		distance = 24.0 - distance
	}
	exponent := -math.Pow(distance, 2) / (2 * math.Pow(curve.Width, 2))
	return curve.Amount * math.Exp(exponent)
}

func (ue *UtilityEngine) GetInterruptAction(actorID string, state *core.SimulationState) string {
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

func (ue *UtilityEngine) CanSafelyWait(actorID string, projectedWait time.Duration, state *core.SimulationState) bool {
	actorMeters, exists := ue.meters[actorID]
	if !exists {
		return false
	}

	var actorObj domain.Actor
	for _, a := range state.Blueprint.Actors {
		if a.ActorID == actorID {
			actorObj = a
			break
		}
	}

	waitHours := projectedWait.Hours()
	const safetyThreshold = 5.0

	for meterID, bio := range actorObj.Biology {
		if bio.DecayPerHour > 0 {
			currentVal := actorMeters[meterID]
			decayAmount := bio.DecayPerHour * waitHours

			for _, m := range state.Blueprint.Meters {
				// We now use the strong geom.CurveType
				if m.MeterID == meterID && m.Curve == geom.Exponential {
					decayAmount *= 1.5
					break
				}
			}

			if (currentVal - decayAmount) < safetyThreshold {
				return false
			}
		}
	}
	return true
}

func (ue *UtilityEngine) InterruptCurrentTask(actorID string, state *core.SimulationState) bool {
	activeActID, exists := ue.activeAction[actorID]
	if !exists {
		return true
	}

	for _, act := range state.Blueprint.Actions {
		if act.ActionID == activeActID {
			if !act.Interruptible {
				return false
			}
			if act.DeviceID != "" {
				if devLedger, ok := state.Devices[act.DeviceID]; ok {
					devLedger.State = domain.DeviceStateStandby // Changed from Off to Standby
				}
			}
			break
		}
	}

	delete(ue.activeAction, actorID)
	delete(ue.activeTasks, actorID)

	if actorLedger, ok := state.Actors[actorID]; ok {
		actorLedger.StateEndsAt = state.SimTime
	}
	return true
}

func (ue *UtilityEngine) ResetMeters(actorID string, startingMeters map[string]float64) {
	if _, exists := ue.meters[actorID]; !exists {
		ue.meters[actorID] = make(map[string]float64)
	}
	for k, v := range startingMeters {
		ue.meters[actorID][k] = v
	}
}

// ApplyModifiersToMeters now accepts the flattened MeterEffect mapping.
func (ue *UtilityEngine) ApplyModifiersToMeters(actorID string, modifiers map[string]domain.MeterEffect, limits map[string]float64) {
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

func (ue *UtilityEngine) HasMeters(actorID string, costs map[string]probability.SampleSpace) bool {
	actorMeters, exists := ue.meters[actorID]
	if !exists {
		return false
	}
	for meterID, costSpace := range costs {
		// Sample the cost right now to see if they can afford it!
		sampledCost := ue.sampler.Sample(costSpace)
		currentVal, ok := actorMeters[meterID]
		if ok && currentVal < sampledCost {
			return false
		}
	}
	return true
}

func (ue *UtilityEngine) GetActorSnapshot(actorID string) core.StateSnapshot {
	actorMeters, exists := ue.meters[actorID]
	if !exists {
		return nil
	}
	snapshot := make(core.StateSnapshot)
	for meterID, value := range actorMeters {
		snapshot["actor."+meterID] = value
	}
	return snapshot
}

func (ue *UtilityEngine) GetActionUrgency(actorID string, actionID string, state *core.SimulationState) float64 {
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

func (ue *UtilityEngine) Process(state *core.SimulationState, snapshot core.StateSnapshot, tickDuration time.Duration) ([]core.ActorTickState, []string, []string) {
	var activeHumanActors []core.ActorTickState
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

		// The Macro-State check!
		if actorLedger.CurrentPhase == domain.PhaseTypeAway {
			continue // Don't process AI logic if they aren't home!
		}

		for meterID, bio := range actor.Biology {
			decayRate := bio.DecayPerHour

			// Use the new CurrentPhase from the ActorLedger instead of calculating it live
			if mult, ok := bio.PhaseMultipliers[string(actorLedger.CurrentPhase)]; ok {
				decayRate *= mult
			}

			decayAmount := decayRate * (tickDuration.Seconds() / 3600.0)

			if decayAmount > 0 {
				for _, mTemplate := range state.Blueprint.Meters {
					if mTemplate.MeterID == meterID && mTemplate.Curve == geom.Exponential {
						currentVal := ue.meters[actor.ActorID][meterID]
						factor := (maxMeters[meterID] - currentVal) / maxMeters[meterID]
						if factor < 0.1 {
							factor = 0.1
						}
						decayAmount *= (1.0 + factor)
						break
					}
				}
			}

			ue.meters[actor.ActorID][meterID] -= decayAmount

			if ue.meters[actor.ActorID][meterID] < 0 {
				ue.meters[actor.ActorID][meterID] = 0
			}
			if limit, ok := maxMeters[meterID]; ok && ue.meters[actor.ActorID][meterID] > limit {
				ue.meters[actor.ActorID][meterID] = limit
			}
		}

		if state.SimTime.Before(actorLedger.StateEndsAt) {
			activeHumanActors = append(activeHumanActors, core.ActorTickState{
				ActorID:  actor.ActorID,
				ActionID: ue.activeAction[actor.ActorID],
			})

			if task, exists := ue.activeTasks[actor.ActorID]; exists {
				elapsedNow := state.SimTime.Sub(task.StartTime)
				elapsedPrev := elapsedNow - tickDuration

				pNow := float64(elapsedNow) / float64(task.TotalDuration)
				pPrev := float64(elapsedPrev) / float64(task.TotalDuration)

				for meterID, fill := range task.Satisfies {
					// Using the pure go-maths library!
					integralNow := geom.CurveIntegral(fill.Curve, pNow)
					integralPrev := geom.CurveIntegral(fill.Curve, pPrev)

					deltaPercentage := integralNow - integralPrev
					if deltaPercentage > 0 {
						ue.meters[actor.ActorID][meterID] += fill.Amount * deltaPercentage
						limit := maxMeters[meterID]
						if ue.meters[actor.ActorID][meterID] > limit {
							ue.meters[actor.ActorID][meterID] = limit
						}
					}
				}
			}
			continue
		}

		if activeAct, wasBusy := ue.activeAction[actor.ActorID]; wasBusy {
			for _, act := range state.Blueprint.Actions {
				if act.ActionID == activeAct && act.DeviceID != "" {
					if devLedger, exists := state.Devices[act.DeviceID]; exists {
						devLedger.State = domain.DeviceStateStandby
					}
					break
				}
			}
			delete(ue.activeAction, actor.ActorID)
			delete(ue.activeTasks, actor.ActorID)
		}

		bestScore := 0.0
		var validActions []domain.ActionTemplate
		actionScores := make(map[string]float64)

		suppressedDeficits := make(map[string]float64)
		hasCommitment := false

		if actorLedger.CurrentCommitment != nil {
			hasCommitment = true
			for _, act := range state.Blueprint.Actions {
				if act.ActionID == actorLedger.CurrentCommitment.ActionID {
					for m, f := range act.Satisfies {
						suppressedDeficits[m] = f.Amount
					}
					break
				}
			}
		}

		for _, template := range state.Blueprint.Actions {
			if hasCommitment && !template.Interruptible {
				continue
			}

			var primaryMeter string
			var maxFill float64 = -1.0
			for m, fill := range template.Satisfies {
				if fill.Amount > maxFill {
					maxFill = fill.Amount
					primaryMeter = m
				}
			}

			if primaryMeter != "" {
				currentVal := ue.meters[actor.ActorID][primaryMeter]
				maxVal := maxMeters[primaryMeter]
				deficit := maxVal - currentVal

				if deficit < (maxVal * 0.1) {
					continue
				}
			}

			if template.DeviceID != "" {
				isDeviceLocked := false

				if state.House.ResourceLocks != nil {
					if _, isLocked := state.House.ResourceLocks[template.DeviceID]; isLocked {
						isDeviceLocked = true
					}
				}
				if devLedger, exists := state.Devices[template.DeviceID]; exists {
					if devLedger.State == domain.DeviceStateOn {
						isDeviceLocked = true
					}
				}

				if isDeviceLocked {
					if template.Sharing == nil || template.Sharing.Type != string(domain.SharingFreeRider) {
						continue
					}
				}
			}

			available := true
			for _, cond := range template.AvailableWhen {
				if !core.EvaluateCondition(cond, snapshot) {
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

				if suppression, exists := suppressedDeficits[meterID]; exists {
					deficit -= suppression
				}

				if deficit <= 0 {
					continue
				}

				fillAmount := fill.Amount
				if fillAmount > deficit {
					fillAmount = deficit
				}

				rawUrgency := deficit / maxMeters[meterID]
				curveType := geom.Linear
				for _, m := range state.Blueprint.Meters {
					if m.MeterID == meterID {
						curveType = m.Curve
						break
					}
				}

				// Evaluate using go-maths library
				urgency := geom.EvaluateCurve(curveType, rawUrgency)
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

		if len(wheel) > 0 {
			rollDist := probability.SampleSpace{Type: probability.UniformDistribution, Min: 0.0, Max: 1.0}
			fVal := ue.sampler.Sample(rollDist)

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
				// Evaluate duration based on the DynamicDistribution
				duration := ue.sampler.SampleDuration(template.Duration.Base)

				// Apply DynamicDistribution condition shifts!
				for _, mod := range template.Duration.Modifiers {
					if core.EvaluateCondition(mod.Condition, snapshot) {
						duration += time.Duration(mod.CompiledTransform.FlatShift)
					}
				}

				if template.DeviceID != "" {
					if devLedger, exists := state.Devices[template.DeviceID]; exists {
						devLedger.State = domain.DeviceStateOn
						devLedger.StateEndsAt = state.SimTime.Add(duration)
					}
				}

				for meterID, costSpace := range template.Costs {
					cost := ue.sampler.Sample(costSpace)
					ue.meters[actor.ActorID][meterID] -= cost
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
				activeHumanActors = append(activeHumanActors, core.ActorTickState{
					ActorID:  actor.ActorID,
					ActionID: actionID,
				})
			}
		}
	}

	return activeHumanActors, anomalies, debugLogs
}
