package factory

import (
	"fmt"
	"strings"

	"github.com/tinywideclouds/go-sim-probability/pkg/generator"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

type GenerationRequest struct {
	ArchetypeID            string
	PersonaRequirements    []PersonaRequirement
	SystemIDs              []string
	RequiredDeviceTags     []string
	RequiredWaterSystemTag string
	RoutineIDs             []string
	AlarmIDs               []string
	EventIDs               []string
}

type HouseholdGenerator struct {
	registry *Registry
	sampler  *generator.Sampler
}

func NewHouseholdGenerator(reg *Registry, samp *generator.Sampler) *HouseholdGenerator {
	return &HouseholdGenerator{
		registry: reg,
		sampler:  samp,
	}
}

func (g *HouseholdGenerator) Generate(req GenerationRequest) (*domain.NodeArchetype, error) {
	// [Step 1 and 2 (Water and Hardware) remain identical... omitted for brevity if needed, but here is the full code]
	var selectedWater domain.WaterSystemTemplate
	var candidateWaters []domain.WaterSystemTemplate

	for _, ws := range g.registry.WaterSystems {
		for _, tag := range ws.Tags {
			if tag == req.RequiredWaterSystemTag {
				candidateWaters = append(candidateWaters, ws.Template)
				break
			}
		}
	}

	if len(candidateWaters) > 0 {
		rollDist := domain.ProbabilityDistribution{Type: domain.DistributionTypeUniform, Min: 0.0, Max: float64(len(candidateWaters))}
		roll, _ := g.sampler.Float64(rollDist)
		idx := int(roll)
		if idx >= len(candidateWaters) {
			idx = len(candidateWaters) - 1
		}
		selectedWater = candidateWaters[idx]
	} else {
		selectedWater = domain.WaterSystemTemplate{
			TankCapacityLiters: 200.0, MainsWaterTempCelsius: 10.0, MaxTankTempCelsius: 60.0, StandbyTemperatureLossTick: 0.005,
		}
	}

	node := &domain.NodeArchetype{
		ArchetypeID:         req.ArchetypeID,
		Description:         "Procedurally generated household",
		BaseTempC:           18.0,
		InsulationDecayRate: 0.1,
		Grid: &domain.GridTemplate{
			NominalVoltage: 230.0, WaveCenter: 230.0, WaveAmplitude: 5.0, PeakHour: 18.0, JitterMin: -1.5, JitterMax: 1.5,
		},
		WaterSystem: &selectedWater,
		Meters: []domain.MeterTemplate{
			{MeterID: "energy", Max: 100.0, BaseDecayPerHour: 4.0, Curve: "linear"},
			{MeterID: "hunger", Max: 100.0, BaseDecayPerHour: 6.0, Curve: "linear"},
			{MeterID: "hygiene", Max: 100.0, BaseDecayPerHour: 2.0, Curve: "linear"},
			{MeterID: "leisure", Max: 100.0, BaseDecayPerHour: 5.0, Curve: "linear"},
		},
	}

	concreteDevices := make(map[string]string)
	for _, tag := range req.RequiredDeviceTags {
		var candidates []CatalogDevice
		for _, dev := range g.registry.Devices {
			for _, dTag := range dev.Tags {
				if dTag == tag {
					candidates = append(candidates, dev)
					break
				}
			}
		}

		if len(candidates) == 0 {
			return nil, fmt.Errorf("no devices found in registry for tag: %s", tag)
		}

		rollDist := domain.ProbabilityDistribution{Type: domain.DistributionTypeUniform, Min: 0.0, Max: float64(len(candidates))}
		roll, _ := g.sampler.Float64(rollDist)
		idx := int(roll)
		if idx >= len(candidates) {
			idx = len(candidates) - 1
		}

		chosen := candidates[idx]
		concreteID := fmt.Sprintf("%s_%d", chosen.ID, len(node.Devices)+1)
		concreteDev := chosen.Template
		concreteDev.DeviceID = concreteID

		node.Devices = append(node.Devices, concreteDev)
		concreteDevices[tag] = concreteID
	}

	// 3. Instantiate Personas via Weighted Pool & Gaussian Biology Sampling
	actorCounter := 1
	for _, pReq := range req.PersonaRequirements {
		count := pReq.Min
		if pReq.Max > pReq.Min {
			dist := domain.ProbabilityDistribution{Type: domain.DistributionTypeUniform, Min: float64(pReq.Min), Max: float64(pReq.Max + 1)}
			roll, _ := g.sampler.Float64(dist)
			count = int(roll)
		}

		// Gather weighted pool with prefix filtering
		var pool []CatalogPersona
		totalWeight := 0
		for _, p := range g.registry.Personas {
			if p.Type == pReq.Type {

				// Apply Allowed Filters
				if len(pReq.AllowedPrefixes) > 0 {
					matched := false
					for _, prefix := range pReq.AllowedPrefixes {
						if strings.HasPrefix(p.ID, prefix) {
							matched = true
							break
						}
					}
					if !matched {
						continue
					}
				}

				// Apply Exclude Filters
				excluded := false
				for _, prefix := range pReq.ExcludePrefixes {
					if strings.HasPrefix(p.ID, prefix) {
						excluded = true
						break
					}
				}
				if excluded {
					continue
				}

				pool = append(pool, p)
				weight := p.Frequency
				if weight <= 0 {
					weight = 10
				}
				totalWeight += weight
			}
		}

		if len(pool) == 0 {
			return nil, fmt.Errorf("no personas found in registry matching type and filters: %s", pReq.Type)
		}

		for i := 0; i < count; i++ {
			rollDist := domain.ProbabilityDistribution{Type: domain.DistributionTypeUniform, Min: 0.0, Max: float64(totalWeight)}
			roll, _ := g.sampler.Float64(rollDist)

			var selected CatalogPersona
			accum := 0.0
			for _, p := range pool {
				weight := p.Frequency
				if weight <= 0 {
					weight = 10
				}
				accum += float64(weight)
				if roll <= accum {
					selected = p
					break
				}
			}

			startingMeters := make(map[string]float64)
			for m, dist := range selected.StartingMeters {
				val, err := g.sampler.Float64(dist)
				if err != nil {
					val = 50.0
				}
				if val > 100.0 {
					val = 100.0
				}
				if val < 0.0 {
					val = 0.0
				}
				startingMeters[m] = val
			}

			biology := make(map[string]domain.InstantiatedBiology)
			for m, bioCfg := range selected.Biology {
				decay, err := g.sampler.Float64(bioCfg.DecayPerHour)
				if err != nil {
					decay = 5.0
				}
				if decay < 0.0 {
					decay = 0.0
				}

				biology[m] = domain.InstantiatedBiology{
					DecayPerHour:     decay,
					PhaseMultipliers: bioCfg.PhaseMultipliers,
				}
			}

			temp := 1.0
			for _, t := range selected.Traits {
				if t == "chaotic" {
					temp += 1.0
				} else if t == "disciplined" {
					temp -= 0.5
				}
			}

			actor := domain.Actor{
				ActorID:            fmt.Sprintf("%s_%d", selected.ID, actorCounter),
				Type:               selected.Type,
				AIModel:            "stable",
				StartingMeters:     startingMeters,
				Biology:            biology,
				Phases:             selected.Phases,
				SoftmaxTemperature: temp,
			}
			node.Actors = append(node.Actors, actor)
			actorCounter++
		}
	}

	// 4. Link Actions to Rolled Hardware
	for _, cAct := range g.registry.Actions {
		concreteID, tagExists := concreteDevices[cAct.RequiresDeviceTag]

		if cAct.RequiresDeviceTag == "" || tagExists {
			actTpl := cAct.Template
			if tagExists {
				actTpl.DeviceID = concreteID
			}
			if len(actTpl.ActorTags) == 0 {
				actTpl.ActorTags = []string{"adult", "child", "elderly"}
			}
			node.Actions = append(node.Actions, actTpl)
		}
	}

	// 5. Inject Ambient Systems
	for _, sid := range req.SystemIDs {
		cs, exists := g.registry.Systems[sid]
		if exists {
			for _, scenario := range cs.Scenarios {
				linkedScenario := scenario
				linkedScenario.Actions = make([]domain.ScenarioAction, len(scenario.Actions))
				copy(linkedScenario.Actions, scenario.Actions)

				for i, act := range linkedScenario.Actions {
					if concreteID, tagExists := concreteDevices[act.DeviceID]; tagExists {
						linkedScenario.Actions[i].DeviceID = concreteID
					}
				}
				node.Scenarios = append(node.Scenarios, linkedScenario)
			}
		}
	}

	for _, rid := range req.RoutineIDs {
		if cRout, exists := g.registry.Routines[rid]; exists {
			node.RoutineTemplates = append(node.RoutineTemplates, cRout.Template)
		}
	}

	for _, aid := range req.AlarmIDs {
		if cAlarm, exists := g.registry.Alarms[aid]; exists {
			node.Alarms = append(node.Alarms, cAlarm.Template)
		}
	}

	for _, eid := range req.EventIDs {
		cEvent, exists := g.registry.CollectiveEvents[eid]
		if !exists {
			continue
		}

		rollDist := domain.ProbabilityDistribution{Type: domain.DistributionTypeUniform, Min: 0.0, Max: 1.0}
		roll, _ := g.sampler.Float64(rollDist)
		if roll > cEvent.Selection.Weight {
			continue
		}

		instantiatedEvent := domain.CollectiveEvent{
			EventID:         cEvent.Template.EventID,
			Action:          cEvent.Template.Action,
			BaseFragility:   cEvent.Template.BaseFragility,
			AbortConditions: cEvent.Template.AbortConditions,
		}

		var leadActorID string
		for _, a := range node.Actors {
			if leadActorID == "" && (a.Type == cEvent.Template.LeadRequirement || cEvent.Template.LeadRequirement == "any") {
				leadActorID = a.ActorID
				break
			}
		}

		if leadActorID == "" {
			continue
		}
		instantiatedEvent.LeadActor = leadActorID

		for _, rule := range cEvent.Template.DependentRules {
			for _, a := range node.Actors {
				if a.ActorID != leadActorID && a.Type == rule.Type {
					depActor := domain.DependentActor{ActorID: a.ActorID, FrictionWeight: rule.FrictionWeight, PatienceLimit: rule.PatienceLimit}
					instantiatedEvent.DependentActors = append(instantiatedEvent.DependentActors, depActor)
				}
			}
		}
		node.CollectiveEvents = append(node.CollectiveEvents, instantiatedEvent)
	}

	return node, nil
}
