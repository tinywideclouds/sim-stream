package aiengine

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/tinywideclouds/go-power-simulator/internal/engine"
	"github.com/tinywideclouds/go-sim-probability/pkg/generator"
	"github.com/tinywideclouds/go-sim-probability/pkg/parsers"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

type TickResult struct {
	Timestamp       time.Time
	GridVoltage     float64
	TotalWatts      float64
	TotalColdLiters float64
	TotalHotLiters  float64
	IndoorTempC     float64
	TankTempC       float64
	ActiveDevices   []string
	ActiveActors    []string
	Anomalies       []string
	DebugLog        []string
}

type AIEngine interface {
	Process(state *engine.SimulationState, snapshot parsers.StateSnapshot, tickDuration time.Duration) ([]string, []string, []string)
	InterruptCurrentTask(actorID string, state *engine.SimulationState) bool
	GetActionUrgency(actorID string, actionID string, state *engine.SimulationState) float64
	ForceTask(actorID string, taskName string, duration time.Duration, startTime time.Time, satisfies map[string]domain.ActionFill)
}

type Orchestrator struct {
	humanAI AIEngine
	sampler *generator.Sampler
}

func NewOrchestrator(ai AIEngine, sampler *generator.Sampler) *Orchestrator {
	return &Orchestrator{
		humanAI: ai,
		sampler: sampler,
	}
}

func (o *Orchestrator) Tick(state *engine.SimulationState, tickDuration time.Duration, weather engine.WeatherProvider, grid engine.GridProvider) TickResult {
	state.SimTime = state.SimTime.Add(tickDuration)

	snapshot := engine.BuildEnvironmentSnapshot(state.SimTime, weather)
	snapshot["indoor_temp_c"] = state.IndoorTempC
	snapshot["tank_temp_c"] = state.HotWaterTankC
	snapshot["time.hour_float"] = float64(state.SimTime.Hour()) + (float64(state.SimTime.Minute()) / 60.0) + (float64(state.SimTime.Second()) / 3600.0)

	o.reapFinishedEvents(state)
	o.processPendingWindows(state)

	activeHumanActors, anomalies, debugLogs := o.humanAI.Process(state, snapshot, tickDuration)

	o.handleSharingIntents(state, activeHumanActors)

	activeAmbientActors := engine.ProcessAmbientSystems(state, snapshot, o.sampler)
	activeActors := append(activeHumanActors, activeAmbientActors...)

	participantCounts := make(map[string]int)
	if state.House.PendingEvents != nil {
		for _, ev := range state.House.PendingEvents {
			if ev.IsExecuting && ev.DeviceID != "" {
				participantCounts[ev.DeviceID] = len(ev.Participants)
			}
		}
	}

	physics := engine.ProcessPhysics(state, tickDuration, grid, participantCounts)

	externalTempC := 5.0
	if val, ok := snapshot["weather.external_temp_c"].(float64); ok {
		externalTempC = val
	}
	engine.ProcessThermodynamics(state, physics.HeaterWatts, externalTempC, tickDuration)

	return TickResult{
		Timestamp:       state.SimTime,
		GridVoltage:     physics.GridVoltage,
		TotalWatts:      physics.TotalWatts,
		TotalColdLiters: physics.ColdLiters,
		TotalHotLiters:  physics.HotLiters,
		IndoorTempC:     state.IndoorTempC,
		TankTempC:       state.HotWaterTankC,
		ActiveDevices:   physics.ActiveDevices,
		ActiveActors:    activeActors,
		Anomalies:       anomalies,
		DebugLog:        debugLogs,
	}
}

func (o *Orchestrator) handleSharingIntents(state *engine.SimulationState, activeHumanActors []string) {
	if state.House.PendingEvents == nil {
		state.House.PendingEvents = make(map[string]*engine.PendingEvent)
	}

	for _, str := range activeHumanActors {
		parts := strings.Split(str, ":")
		if len(parts) != 2 {
			continue
		}
		actorID, actionID := parts[0], parts[1]

		var template *domain.ActionTemplate
		for _, act := range state.Blueprint.Actions {
			if act.ActionID == actionID {
				template = &act
				break
			}
		}
		if template == nil || template.Sharing == nil || template.Sharing.Type == domain.SharingStrictMutex {
			continue
		}

		eventFound := false
		for _, ev := range state.House.PendingEvents {
			if ev.ActionID == actionID {
				for _, p := range ev.Participants {
					if p == actorID {
						eventFound = true
						break
					}
				}
			}
			if eventFound {
				break
			}
		}
		if eventFound {
			continue
		}

		eventID := fmt.Sprintf("%s_%s_%d", actionID, actorID, state.SimTime.Unix())
		gatheringDuration, _ := time.ParseDuration(template.Sharing.GatheringWindow)

		var endsAt time.Time
		if ledger, ok := state.Actors[actorID]; ok {
			endsAt = ledger.StateEndsAt
		} else {
			endsAt = state.SimTime.Add(15 * time.Minute)
		}

		ev := &engine.PendingEvent{
			EventID:         eventID,
			ActionID:        actionID,
			DeviceID:        template.DeviceID,
			InitiatorID:     actorID,
			Participants:    []string{actorID},
			GatheringEndsAt: state.SimTime.Add(gatheringDuration),
			IsExecuting:     false,
		}

		if template.Sharing.Type == domain.SharingFreeRider {
			ev.IsExecuting = true
			ev.GatheringEndsAt = endsAt
		} else if gatheringDuration > 0 {

			slog.Info("GATHERING WINDOW OPENED", "event", eventID, "action", actionID, "initiator", actorID, "duration", gatheringDuration)

			o.humanAI.InterruptCurrentTask(actorID, state)

			if ledger, ok := state.Actors[actorID]; ok {
				ledger.CurrentCommitment = &engine.Commitment{
					ActionID:  actionID,
					Role:      "lead",
					ExpiresAt: ev.GatheringEndsAt.Add(2 * time.Hour),
				}
			}

			if template.DeviceID != "" {
				if devLedger, ok := state.Devices[template.DeviceID]; ok {
					devLedger.State = domain.DeviceStateOff
				}
				if state.House.ResourceLocks == nil {
					state.House.ResourceLocks = make(map[string]string)
				}
				state.House.ResourceLocks[template.DeviceID] = eventID
			}
		}

		state.House.PendingEvents[eventID] = ev
	}
}

func (o *Orchestrator) processPendingWindows(state *engine.SimulationState) {
	for _, ev := range state.House.PendingEvents {
		var template *domain.ActionTemplate
		for _, act := range state.Blueprint.Actions {
			if act.ActionID == ev.ActionID {
				template = &act
				break
			}
		}
		if template == nil {
			continue
		}

		if ev.IsExecuting {
			if template.Sharing.Type == domain.SharingFreeRider {
				o.inviteAndJoin(state, ev, template)
			}
			continue
		}

		o.inviteAndJoin(state, ev, template)

		if state.SimTime.After(ev.GatheringEndsAt) || state.SimTime.Equal(ev.GatheringEndsAt) {
			ev.IsExecuting = true

			slog.Info("SHARED EVENT EXECUTING", "event", ev.EventID, "participants", len(ev.Participants), "action", ev.ActionID)

			duration, err := o.sampler.Duration(template.Duration)
			if err != nil {
				duration = 15 * time.Minute
			}

			if template.Sharing.Type == domain.SharingScalable {
				multiplier := 1.0 + float64(len(ev.Participants)-1)*template.Sharing.DurationMultiplierPerActor
				duration = time.Duration(float64(duration) * multiplier)
			}

			for _, participantID := range ev.Participants {
				o.humanAI.InterruptCurrentTask(participantID, state)
				o.humanAI.ForceTask(participantID, ev.ActionID, duration, state.SimTime, template.Satisfies)

				if ledger, ok := state.Actors[participantID]; ok {
					ledger.CurrentCommitment = nil
					ledger.StateEndsAt = state.SimTime.Add(duration)
				}
			}

			if ev.DeviceID != "" {
				if devLedger, ok := state.Devices[ev.DeviceID]; ok {
					devLedger.State = domain.DeviceStateOn
					devLedger.StateEndsAt = state.SimTime.Add(duration)
				}
			}

			ev.GatheringEndsAt = state.SimTime.Add(duration)
		}
	}
}

func (o *Orchestrator) inviteAndJoin(state *engine.SimulationState, ev *engine.PendingEvent, template *domain.ActionTemplate) {
	if template.Sharing.MaxParticipants > 0 && len(ev.Participants) >= template.Sharing.MaxParticipants {
		return
	}

	for _, actor := range state.Blueprint.Actors {
		if actor.AIModel != "utility" {
			continue
		}

		isParticipating := false
		for _, p := range ev.Participants {
			if p == actor.ActorID {
				isParticipating = true
				break
			}
		}
		if isParticipating {
			continue
		}

		hasPermission := false
		if len(template.ActorTags) == 0 {
			hasPermission = true
		} else {
			for _, tag := range template.ActorTags {
				if tag == actor.Type || tag == "any" {
					hasPermission = true
					break
				}
			}
		}
		if !hasPermission {
			continue
		}

		urgency := o.humanAI.GetActionUrgency(actor.ActorID, ev.ActionID, state)
		if urgency > 0.1 {

			slog.Info("ACTOR JOINED SHARED EVENT", "event", ev.EventID, "actor", actor.ActorID, "action", ev.ActionID)
			ev.Participants = append(ev.Participants, actor.ActorID)

			if !ev.IsExecuting {
				if ledger, ok := state.Actors[actor.ActorID]; ok {
					ledger.CurrentCommitment = &engine.Commitment{
						ActionID:  ev.ActionID,
						Role:      "participant",
						ExpiresAt: ev.GatheringEndsAt.Add(2 * time.Hour),
					}
				}
			} else if template.Sharing.Type == domain.SharingFreeRider {
				o.humanAI.InterruptCurrentTask(actor.ActorID, state)
				durationRemaining := ev.GatheringEndsAt.Sub(state.SimTime)
				if durationRemaining < 0 {
					durationRemaining = 0
				}
				o.humanAI.ForceTask(actor.ActorID, ev.ActionID, durationRemaining, state.SimTime, template.Satisfies)
				if ledger, ok := state.Actors[actor.ActorID]; ok {
					ledger.StateEndsAt = state.SimTime.Add(durationRemaining)
				}
			}

			if template.Sharing.MaxParticipants > 0 && len(ev.Participants) >= template.Sharing.MaxParticipants {
				return
			}
		}
	}
}

func (o *Orchestrator) reapFinishedEvents(state *engine.SimulationState) {
	for id, ev := range state.House.PendingEvents {
		if ev.IsExecuting {
			if state.SimTime.After(ev.GatheringEndsAt) || state.SimTime.Equal(ev.GatheringEndsAt) {
				delete(state.House.PendingEvents, id)
				if ev.DeviceID != "" {
					if lock, ok := state.House.ResourceLocks[ev.DeviceID]; ok && lock == id {
						delete(state.House.ResourceLocks, ev.DeviceID)
					}
				}
				for _, participantID := range ev.Participants {
					if ledger, ok := state.Actors[participantID]; ok {
						if ledger.CurrentCommitment != nil && ledger.CurrentCommitment.ActionID == ev.ActionID {
							ledger.CurrentCommitment = nil
						}
					}
				}
			}
		} else {
			if ledger, ok := state.Actors[ev.InitiatorID]; ok {
				if ledger.CurrentCommitment == nil || ledger.CurrentCommitment.ActionID != ev.ActionID {
					for _, pID := range ev.Participants {
						if pLedger, pOk := state.Actors[pID]; pOk {
							if pLedger.CurrentCommitment != nil && pLedger.CurrentCommitment.ActionID == ev.ActionID {
								pLedger.CurrentCommitment = nil
							}
						}
					}
					delete(state.House.PendingEvents, id)
					if ev.DeviceID != "" {
						if lock, ok := state.House.ResourceLocks[ev.DeviceID]; ok && lock == id {
							delete(state.House.ResourceLocks, ev.DeviceID)
						}
					}
				}
			}
		}
	}
}
