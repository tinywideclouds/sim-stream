// aiengine/utility_engine.go
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

func evaluateGaussianBonus(curve domain.UtilityBonusCurve, currentValue float64) float64 {
	distance := math.Abs(currentValue - curve.Peak)
	if curve.ContextKey == "time.hour_float" && distance > 12.0 {
		distance = 24.0 - distance
	}
	exponent := -math.Pow(distance, 2) / (2 * math.Pow(curve.Width, 2))
	return curve.Magnitude * math.Exp(exponent)
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

func (ue *UtilityEngine) Process(state *engine.SimulationState, snapshot parsers.EnvironmentSnapshot, tickDuration time.Duration) ([]string, []string, []string) {
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
		if actor.AIModel != "utility" {
			continue
		}

		if _, exists := ue.meters[actor.ActorID]; !exists {
			ue.meters[actor.ActorID] = make(map[string]float64)
			for k, v := range actor.StartingMeters {
				ue.meters[actor.ActorID][k] = v
			}
		}

		actorLedger := state.Actors[actor.ActorID]

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

				// Find the curve type from the Blueprint
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

			// INITIATION FRICTION: Applied only if they are not already doing the action
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
			rollDist := domain.ProbabilityDistribution{
				Type: domain.DistributionTypeUniform,
				Min:  0.0,
				Max:  1.0,
			}
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
			}
		}
	}

	return activeHumanActors, anomalies, debugLogs
}

// applyUrgencyCurve changes how an AI perceives a deficit based on the meter type.
func applyUrgencyCurve(curve string, p float64) float64 {
	if p <= 0.0 {
		return 0.0
	}
	if p >= 1.0 {
		return 1.0
	}

	switch curve {
	case "front_loaded":
		// Spikes early (at low deficit), then plateaus. Great for Hygiene.
		return 1.0 - math.Pow(1.0-p, 3)
	case "exponential", "back_loaded":
		// Ignores the deficit until it's huge, then panics. Great for Hunger.
		return math.Pow(p, 3)
	case "s_curve":
		// Deadzone at start, massive spike in the middle, plateau at end.
		return p * p * (3.0 - 2.0*p)
	case "linear":
		fallthrough
	default:
		return p
	}
}
