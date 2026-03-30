package engine

import (
	"time"

	"github.com/tinywideclouds/go-sim-physics/pkg/thermal"
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
}

type Orchestrator struct {
	scheduler    *Scheduler
	negotiator   *Negotiator
	executor     *Executor
	rolloverHour int
	dailyPlan    map[string]map[string]ScheduledRoutine
	currentDay   time.Time
}

func NewOrchestrator(s *Scheduler, n *Negotiator, e *Executor, rolloverHour int) *Orchestrator {
	return &Orchestrator{
		scheduler:    s,
		negotiator:   n,
		executor:     e,
		rolloverHour: rolloverHour,
		dailyPlan:    make(map[string]map[string]ScheduledRoutine),
	}
}

func (o *Orchestrator) Tick(state *SimulationState, tickDuration time.Duration, weather WeatherProvider, grid GridProvider) TickResult {
	state.SimTime = state.SimTime.Add(tickDuration)

	snap := BuildEnvironmentSnapshot(state.SimTime, weather)
	snap["indoor_temp_c"] = state.IndoorTempC
	snap["tank_temp_c"] = state.HotWaterTankC // NEW: Expose the tank temp to the AI!

	outsideTempC := 5.0
	if val, ok := snap["weather.external_temp_c"].(float64); ok {
		outsideTempC = val
	}

	logicalDay := state.SimTime
	if state.SimTime.Hour() < o.rolloverHour {
		logicalDay = state.SimTime.Add(-24 * time.Hour)
	}
	midnightOfLogicalDay := logicalDay.Truncate(24 * time.Hour)

	if !o.currentDay.Equal(midnightOfLogicalDay) || len(o.dailyPlan) == 0 {
		o.currentDay = midnightOfLogicalDay
		o.BuildDailyPlan(state, midnightOfLogicalDay, snap)
	}

	var activeActors []string

	// 3A. HUMAN AI LOOP
	for _, actorTpl := range state.Blueprint.Actors {
		actorLedger := state.Actors[actorTpl.ActorID]
		actorPlan := o.dailyPlan[actorTpl.ActorID]

		if actorLedger.CurrentState == domain.ActorStateAsleep || actorLedger.CurrentState == domain.ActorStateHomeFree {
			for routineID, routinePlan := range actorPlan {
				if !routinePlan.HasStarted && !state.SimTime.Before(routinePlan.TargetStart) && state.SimTime.Before(routinePlan.TargetDeadline) {
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
			var tpl *domain.RoutineTemplate
			for _, rt := range state.Blueprint.RoutineTemplates {
				if rt.RoutineID == actorLedger.CurrentRoutineID {
					tpl = &rt
					break
				}
			}
			if tpl != nil {
				_ = o.executor.AdvanceRoutine(state, actorTpl.ActorID, tpl, routinePlan.TargetDeadline)
			}

			if actorLedger.CurrentState == domain.ActorStateRoutineActive {
				taskName := "transitioning"
				if tpl != nil {
					idx := actorLedger.RoutineStepIndex - 1
					if idx >= 0 && idx < len(tpl.Tasks) {
						taskName = tpl.Tasks[idx]
					}
				}
				activeActors = append(activeActors, actorTpl.ActorID+":"+taskName)
			}
		}
	}

	// 3B. AMBIENT AI LOOP
	for _, scenario := range state.Blueprint.Scenarios {
		if scenario.Trigger == nil || scenario.Trigger.Type != 2 {
			continue
		}

		conditionsMet := true
		for _, cond := range scenario.Trigger.BaseConditions {
			pass, err := parsers.CheckCondition(cond, snap)
			if err != nil || !pass {
				conditionsMet = false
				break
			}
		}

		if conditionsMet {
			var triggeringActor *ActorLedger
			var triggeringActorID string

			for _, tag := range scenario.ActorTags {
				for _, actorTpl := range state.Blueprint.Actors {
					if actorTpl.Type == tag {
						ledger := state.Actors[actorTpl.ActorID]
						if state.SimTime.After(ledger.Satiety[scenario.ScenarioID]) {
							triggeringActor = ledger
							triggeringActorID = actorTpl.ActorID
							break
						}
					}
				}
				if triggeringActor != nil {
					break
				}
			}

			if triggeringActor != nil {
				var maxDuration time.Duration
				for _, action := range scenario.Actions {
					durDist, ok := action.Parameters["duration"]
					var actionDur time.Duration
					if ok {
						actionDur, _ = o.executor.sampler.Duration(durDist)
					}

					if devLedger, exists := state.Devices[action.DeviceID]; exists {
						devLedger.State = action.State
						devLedger.StateEndsAt = state.SimTime.Add(actionDur)
					}
					if actionDur > maxDuration {
						maxDuration = actionDur
					}
				}

				if scenario.Trigger.FatigueRule.LockoutDuration != "" {
					lockout, _ := time.ParseDuration(scenario.Trigger.FatigueRule.LockoutDuration)
					triggeringActor.Satiety[scenario.ScenarioID] = state.SimTime.Add(lockout)
				}

				activeActors = append(activeActors, triggeringActorID+":"+scenario.ScenarioID)
			}
		}
	}

	// 4. PHYSICS LOOP
	var totalWatts, spaceHeaterWatts, waterHeaterWatts, coldLiters, hotLiters float64
	var activeDeviceIDs []string

	nominalVoltage := grid.NominalVoltage()
	currentVoltage := grid.LiveVoltage(state.SimTime)
	voltageMultiplier := (currentVoltage * currentVoltage) / (nominalVoltage * nominalVoltage)

	for _, devTpl := range state.Blueprint.Devices {
		ledger := state.Devices[devTpl.DeviceID]

		if ledger.State == domain.DeviceStateOn && !state.SimTime.Before(ledger.StateEndsAt) {
			ledger.State = domain.DeviceStateStandby
		}

		if ledger.State == domain.DeviceStateOn {
			activeDeviceIDs = append(activeDeviceIDs, devTpl.DeviceID)

			actualWatts := devTpl.ElectricalProfile.MaxWatts * voltageMultiplier
			totalWatts += actualWatts

			if devTpl.Taxonomy.Category == domain.DeviceCategoryHeating { // 2 = Space Heating
				spaceHeaterWatts += actualWatts
			} else if devTpl.Taxonomy.Category == 6 { // 6 = Water Heating
				waterHeaterWatts += actualWatts
			}

			if devTpl.WaterProfile != nil {
				mins := tickDuration.Minutes()
				drawnCold := devTpl.WaterProfile.ColdLitersPerMinute * mins
				drawnHot := devTpl.WaterProfile.HotLitersPerMinute * mins

				coldLiters += drawnCold
				hotLiters += drawnHot

				// Tank Cooling Dynamics (Mixing cold mains water)
				if drawnHot > 0 {
					tankCap := 150.0
					streetTemp := 10.0
					remaining := tankCap - drawnHot
					if remaining < 0 {
						remaining = 0
					}
					state.HotWaterTankC = ((remaining * state.HotWaterTankC) + (drawnHot * streetTemp)) / tankCap
				}
			}
		} else if ledger.State == domain.DeviceStateStandby {
			standbyWatts := devTpl.ElectricalProfile.StandbyWatts * voltageMultiplier
			totalWatts += standbyWatts
		}
	}

	// NEW: Tank Heating Dynamics (Joule Heating Equation)
	tankCapLiters := 150.0
	if waterHeaterWatts > 0 {
		// Q = m * c * deltaT  =>  deltaT = Q / (m * c)
		joules := waterHeaterWatts * tickDuration.Seconds()
		deltaT := joules / (tankCapLiters * 4184.0)
		state.HotWaterTankC += deltaT

		// Safety limit so it doesn't boil
		if state.HotWaterTankC > 60.0 {
			state.HotWaterTankC = 60.0
		}
	}

	// NEW: Passive Tank Heat Loss (Simulation of insulation bleed)
	// Loses approx 0.5°C per hour to the ambient air
	state.HotWaterTankC -= 0.002

	// 5. THERMAL LOOP (Space Heating)
	thermalProps := thermal.ThermalProperties{
		InsulationDecayRate: state.Blueprint.InsulationDecayRate,
		HeatingEfficiency:   0.001,
	}
	state.IndoorTempC = thermal.NextTemperature(thermalProps, state.IndoorTempC, outsideTempC, spaceHeaterWatts, tickDuration)

	return TickResult{
		Timestamp:       state.SimTime,
		GridVoltage:     currentVoltage,
		TotalWatts:      totalWatts,
		TotalColdLiters: coldLiters,
		TotalHotLiters:  hotLiters,
		IndoorTempC:     state.IndoorTempC,
		TankTempC:       state.HotWaterTankC,
		ActiveDevices:   activeDeviceIDs,
		ActiveActors:    activeActors,
	}
}

func (o *Orchestrator) BuildDailyPlan(state *SimulationState, midnight time.Time, snap parsers.EnvironmentSnapshot) {
	for _, a := range state.Blueprint.Actors {
		plans, _ := o.scheduler.ScheduleDay(a, midnight, snap)
		o.dailyPlan[a.ActorID] = make(map[string]ScheduledRoutine)
		for _, p := range plans {
			o.dailyPlan[a.ActorID][p.RoutineID] = p
		}
	}

	for _, ce := range state.Blueprint.CollectiveEvents {
		leadPlan, exists := o.dailyPlan[ce.LeadActor][ce.Action]
		if !exists {
			continue
		}
		leadPart := Participant{ActorID: ce.LeadActor, TargetTime: leadPlan.TargetDeadline, Weight: 1.0}

		var deps []Participant
		for _, d := range ce.DependentActors {
			depPlan := o.dailyPlan[d.ActorID][ce.Action]
			patience, _ := time.ParseDuration(d.PatienceLimit)
			deps = append(deps, Participant{
				ActorID: d.ActorID, TargetTime: depPlan.TargetDeadline,
				Weight: d.FrictionWeight, PatienceLimit: patience,
			})
		}

		finalDeadline := o.negotiator.ResolveEventTime(leadPart, deps)

		leadPlan.TargetDeadline = finalDeadline
		o.dailyPlan[ce.LeadActor][ce.Action] = leadPlan
		for _, d := range ce.DependentActors {
			depPlan := o.dailyPlan[d.ActorID][ce.Action]
			depPlan.TargetDeadline = finalDeadline
			o.dailyPlan[d.ActorID][ce.Action] = depPlan
		}
	}
}
