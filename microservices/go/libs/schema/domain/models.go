package domain

import (
	pb "github.com/tinywideclouds/go-sim-schema/v1"
)

// ---------------------------------------------------------
// ENUMS (Native Go Enums)
// ---------------------------------------------------------

type DistributionType int

const (
	DistributionTypeUnspecified DistributionType = iota
	DistributionTypeNormal
	DistributionTypeUniform
	DistributionTypeConstant
)

type DeviceCategory int

const (
	DeviceCategoryUnspecified DeviceCategory = iota
	DeviceCategoryCooking
	DeviceCategoryHeating
	DeviceCategoryColdStorage
	DeviceCategoryLighting
	DeviceCategoryWetAppliance
	DeviceCategoryElectronics
	DeviceCategoryBaseLoad
)

type ProfileType int

const (
	ProfileTypeUnspecified ProfileType = iota
	ProfileTypeConstant
	ProfileTypeCyclic
	ProfileTypeVariable
)

type ConditionOperator int

const (
	ConditionOperatorUnspecified ConditionOperator = iota
	ConditionOperatorEq
	ConditionOperatorNeq
	ConditionOperatorGt
	ConditionOperatorLt
	ConditionOperatorGte
	ConditionOperatorLte
)

type DeviceState int

const (
	DeviceStateUnspecified DeviceState = iota
	DeviceStateOn
	DeviceStateOff
	DeviceStateStandby
)

type TriggerType int

const (
	TriggerTypeUnspecified TriggerType = iota
	TriggerTypeTimeOfDay
	TriggerTypeEventReaction
)

// ---------------------------------------------------------
// NATIVE DOMAIN STRUCTS
// ---------------------------------------------------------

type EngineCondition struct {
	ContextKey string            `yaml:"context_key"`
	Operator   ConditionOperator `yaml:"operator"`
	Value      string            `yaml:"value"`
}

type DistributionModifier struct {
	Condition  EngineCondition `yaml:"condition"`
	ShiftMean  string          `yaml:"shift_mean"`
	ShiftValue string          `yaml:"shift_value"`
}

type ProbabilityDistribution struct {
	Type      DistributionType       `yaml:"type"`
	Timeframe string                 `yaml:"timeframe"`
	Value     string                 `yaml:"value"`
	Mean      string                 `yaml:"mean"`
	StdDev    string                 `yaml:"std_dev"`
	Min       float64                `yaml:"min"`
	Max       float64                `yaml:"max"`
	Modifiers []DistributionModifier `yaml:"modifiers"`
}

type DeviceProfile struct {
	Type             ProfileType `yaml:"type"`
	MaxWatts         float64     `yaml:"max_watts"`
	StandbyWatts     float64     `yaml:"standby_watts"`
	CooldownDuration string      `yaml:"cooldown_duration"`
}

type DeviceTaxonomy struct {
	Category  DeviceCategory `yaml:"category"`
	ClassName string         `yaml:"class_name"`
}

type DeviceTemplate struct {
	DeviceID          string            `yaml:"device_id"`
	Taxonomy          DeviceTaxonomy    `yaml:"taxonomy"`
	Specifics         map[string]string `yaml:"specifics"`
	ElectricalProfile DeviceProfile     `yaml:"electrical_profile"`
}

type ActorTemplate struct {
	ActorID string `yaml:"actor_id"`
	Type    string `yaml:"type"`
}

type FatigueRule struct {
	LockoutDuration   string  `yaml:"lockout_duration"`
	RecoveryDuration  string  `yaml:"recovery_duration"`
	PenaltyMultiplier float64 `yaml:"penalty_multiplier"`
}

type Trigger struct {
	Type           TriggerType             `yaml:"type"`
	Distribution   ProbabilityDistribution `yaml:"distribution"`
	BaseConditions []EngineCondition       `yaml:"base_conditions"`
	FatigueRule    FatigueRule             `yaml:"fatigue_rule"`
}

type ScenarioAction struct {
	DeviceID       string                             `yaml:"device_id"`
	State          DeviceState                        `yaml:"state"`
	DelayFromStart string                             `yaml:"delay_from_start"`
	Parameters     map[string]ProbabilityDistribution `yaml:"parameters"`
}

type ScenarioTemplate struct {
	ScenarioID string           `yaml:"scenario_id"`
	ActorID    string           `yaml:"actor_id"`
	Trigger    Trigger          `yaml:"trigger"`
	Actions    []ScenarioAction `yaml:"actions"`
}

type NodeArchetype struct {
	ArchetypeID         string             `yaml:"archetype_id"`
	Description         string             `yaml:"description"`
	BaseTempC           float64            `yaml:"base_temp_c"`
	InsulationDecayRate float64            `yaml:"insulation_decay_rate"`
	Actors              []ActorTemplate    `yaml:"actors"`
	Devices             []DeviceTemplate   `yaml:"devices"`
	Scenarios           []ScenarioTemplate `yaml:"scenarios"`
}

// ---------------------------------------------------------
// MAPPING FUNCTIONS (From Proto)
// ---------------------------------------------------------

func NodeArchetypeFromProto(p *pb.NodeArchetype) *NodeArchetype {
	if p == nil {
		return nil
	}

	node := &NodeArchetype{
		ArchetypeID:         p.ArchetypeId,
		Description:         p.Description,
		BaseTempC:           p.BaseTempC,
		InsulationDecayRate: p.InsulationDecayRate,
		Actors:              make([]ActorTemplate, len(p.Actors)),
		Devices:             make([]DeviceTemplate, len(p.Devices)),
		Scenarios:           make([]ScenarioTemplate, len(p.Scenarios)),
	}

	for i, a := range p.Actors {
		node.Actors[i] = ActorTemplate{
			ActorID: a.ActorId,
			Type:    a.Type,
		}
	}

	for i, d := range p.Devices {
		if d != nil {
			node.Devices[i] = *DeviceTemplateFromProto(d)
		}
	}

	for i, s := range p.Scenarios {
		if s != nil {
			node.Scenarios[i] = *ScenarioTemplateFromProto(s)
		}
	}

	return node
}

func ScenarioTemplateFromProto(p *pb.ScenarioTemplate) *ScenarioTemplate {
	if p == nil {
		return nil
	}

	st := &ScenarioTemplate{
		ScenarioID: p.ScenarioId,
		ActorID:    p.ActorId,
		Actions:    make([]ScenarioAction, len(p.Actions)),
	}

	if p.Trigger != nil {
		st.Trigger = *TriggerFromProto(p.Trigger)
	}

	for i, a := range p.Actions {
		if a != nil {
			st.Actions[i] = *ScenarioActionFromProto(a)
		}
	}

	return st
}

func TriggerFromProto(p *pb.Trigger) *Trigger {
	if p == nil {
		return nil
	}

	t := &Trigger{
		Type:           TriggerType(p.Type),
		BaseConditions: make([]EngineCondition, len(p.BaseConditions)),
	}

	if p.Distribution != nil {
		t.Distribution = *ProbabilityDistributionFromProto(p.Distribution)
	}

	if p.FatigueRule != nil {
		t.FatigueRule = FatigueRule{
			LockoutDuration:   p.FatigueRule.LockoutDuration,
			RecoveryDuration:  p.FatigueRule.RecoveryDuration,
			PenaltyMultiplier: p.FatigueRule.PenaltyMultiplier,
		}
	}

	for i, bc := range p.BaseConditions {
		if bc != nil {
			t.BaseConditions[i] = EngineCondition{
				ContextKey: bc.ContextKey,
				Operator:   ConditionOperator(bc.Operator),
				Value:      bc.Value,
			}
		}
	}

	return t
}

func ScenarioActionFromProto(p *pb.ScenarioAction) *ScenarioAction {
	if p == nil {
		return nil
	}

	sa := &ScenarioAction{
		DeviceID:       p.DeviceId,
		State:          DeviceState(p.State),
		DelayFromStart: p.DelayFromStart,
		Parameters:     make(map[string]ProbabilityDistribution, len(p.Parameters)),
	}

	for k, v := range p.Parameters {
		if v != nil {
			sa.Parameters[k] = *ProbabilityDistributionFromProto(v)
		}
	}

	return sa
}

func ProbabilityDistributionFromProto(p *pb.ProbabilityDistribution) *ProbabilityDistribution {
	if p == nil {
		return nil
	}

	dist := &ProbabilityDistribution{
		Type:   DistributionType(p.Type),
		Value:  p.Value,
		Mean:   p.Mean,
		StdDev: p.StdDev,
		Min:    p.Min,
		Max:    p.Max,
	}

	if len(p.Modifiers) > 0 {
		dist.Modifiers = make([]DistributionModifier, len(p.Modifiers))
		for i, mod := range p.Modifiers {
			if mod != nil && mod.Condition != nil {
				dist.Modifiers[i] = DistributionModifier{
					Condition: EngineCondition{
						ContextKey: mod.Condition.ContextKey,
						Operator:   ConditionOperator(mod.Condition.Operator),
						Value:      mod.Condition.Value,
					},
					ShiftMean:  mod.ShiftMean,
					ShiftValue: mod.ShiftValue,
				}
			}
		}
	}

	return dist
}

func DeviceTemplateFromProto(p *pb.DeviceTemplate) *DeviceTemplate {
	if p == nil {
		return nil
	}

	dt := &DeviceTemplate{
		DeviceID: p.DeviceId,
		Taxonomy: DeviceTaxonomy{
			Category:  DeviceCategory(p.Taxonomy.Category),
			ClassName: p.Taxonomy.ClassName,
		},
		Specifics: make(map[string]string, len(p.Specifics)),
		ElectricalProfile: DeviceProfile{
			Type:             ProfileType(p.ElectricalProfile.Type),
			MaxWatts:         p.ElectricalProfile.MaxWatts,
			StandbyWatts:     p.ElectricalProfile.StandbyWatts,
			CooldownDuration: p.ElectricalProfile.CooldownDuration,
		},
	}

	for k, v := range p.Specifics {
		dt.Specifics[k] = v
	}

	return dt
}

// ---------------------------------------------------------
// MAPPING FUNCTIONS (To Proto)
// ---------------------------------------------------------

func (n *NodeArchetype) ToProto() *pb.NodeArchetype {
	if n == nil {
		return nil
	}

	p := &pb.NodeArchetype{
		ArchetypeId:         n.ArchetypeID,
		Description:         n.Description,
		BaseTempC:           n.BaseTempC,
		InsulationDecayRate: n.InsulationDecayRate,
		Actors:              make([]*pb.ActorTemplate, len(n.Actors)),
		Devices:             make([]*pb.DeviceTemplate, len(n.Devices)),
		Scenarios:           make([]*pb.ScenarioTemplate, len(n.Scenarios)),
	}

	for i, a := range n.Actors {
		p.Actors[i] = &pb.ActorTemplate{
			ActorId: a.ActorID,
			Type:    a.Type,
		}
	}

	for i, d := range n.Devices {
		p.Devices[i] = d.ToProto()
	}

	for i, s := range n.Scenarios {
		p.Scenarios[i] = s.ToProto()
	}

	return p
}

func (s *ScenarioTemplate) ToProto() *pb.ScenarioTemplate {
	if s == nil {
		return nil
	}

	p := &pb.ScenarioTemplate{
		ScenarioId: s.ScenarioID,
		ActorId:    s.ActorID,
		Trigger:    s.Trigger.ToProto(),
		Actions:    make([]*pb.ScenarioAction, len(s.Actions)),
	}

	for i, a := range s.Actions {
		p.Actions[i] = a.ToProto()
	}

	return p
}

func (t *Trigger) ToProto() *pb.Trigger {
	if t == nil {
		return nil
	}

	p := &pb.Trigger{
		Type:           pb.TriggerType(t.Type),
		Distribution:   t.Distribution.ToProto(),
		BaseConditions: make([]*pb.EngineCondition, len(t.BaseConditions)),
		FatigueRule: &pb.FatigueRule{
			LockoutDuration:   t.FatigueRule.LockoutDuration,
			RecoveryDuration:  t.FatigueRule.RecoveryDuration,
			PenaltyMultiplier: t.FatigueRule.PenaltyMultiplier,
		},
	}

	for i, bc := range t.BaseConditions {
		p.BaseConditions[i] = &pb.EngineCondition{
			ContextKey: bc.ContextKey,
			Operator:   pb.ConditionOperator(bc.Operator),
			Value:      bc.Value,
		}
	}

	return p
}

func (sa *ScenarioAction) ToProto() *pb.ScenarioAction {
	if sa == nil {
		return nil
	}

	p := &pb.ScenarioAction{
		DeviceId:       sa.DeviceID,
		State:          pb.DeviceState(sa.State),
		DelayFromStart: sa.DelayFromStart,
		Parameters:     make(map[string]*pb.ProbabilityDistribution, len(sa.Parameters)),
	}

	for k, v := range sa.Parameters {
		p.Parameters[k] = v.ToProto()
	}

	return p
}

func (pd *ProbabilityDistribution) ToProto() *pb.ProbabilityDistribution {
	if pd == nil {
		return nil
	}

	p := &pb.ProbabilityDistribution{
		Type:      pb.DistributionType(pd.Type),
		Value:     pd.Value,
		Mean:      pd.Mean,
		StdDev:    pd.StdDev,
		Min:       pd.Min,
		Max:       pd.Max,
		Modifiers: make([]*pb.DistributionModifier, len(pd.Modifiers)),
	}

	for i, mod := range pd.Modifiers {
		p.Modifiers[i] = &pb.DistributionModifier{
			Condition: &pb.EngineCondition{
				ContextKey: mod.Condition.ContextKey,
				Operator:   pb.ConditionOperator(mod.Condition.Operator),
				Value:      mod.Condition.Value,
			},
			ShiftMean:  mod.ShiftMean,
			ShiftValue: mod.ShiftValue,
		}
	}

	return p
}

func (d *DeviceTemplate) ToProto() *pb.DeviceTemplate {
	if d == nil {
		return nil
	}

	p := &pb.DeviceTemplate{
		DeviceId: d.DeviceID,
		Taxonomy: &pb.DeviceTaxonomy{
			Category:  pb.DeviceCategory(d.Taxonomy.Category),
			ClassName: d.Taxonomy.ClassName,
		},
		Specifics: make(map[string]string, len(d.Specifics)),
		ElectricalProfile: &pb.DeviceProfile{
			Type:             pb.ProfileType(d.ElectricalProfile.Type),
			MaxWatts:         d.ElectricalProfile.MaxWatts,
			StandbyWatts:     d.ElectricalProfile.StandbyWatts,
			CooldownDuration: d.ElectricalProfile.CooldownDuration,
		},
	}

	for k, v := range d.Specifics {
		p.Specifics[k] = v
	}

	return p
}
