package domain

import (
	"time"

	"github.com/tinywideclouds/go-maths/pkg/geom"
	"github.com/tinywideclouds/go-maths/pkg/probability"
	"gopkg.in/yaml.v3"
)

// EngineCondition evaluates an environmental variable against a target.
type EngineCondition struct {
	ContextKey string            `yaml:"context_key"`
	Operator   ConditionOperator `yaml:"operator"`
	Value      string            `yaml:"value"`
}

// DistributionModifier is the YAML-facing struct.
// It allows users to write human-readable strings like `shift_mean: "30m"`.
type DistributionModifier struct {
	Condition        EngineCondition `yaml:"condition"`
	ShiftMean        string          `yaml:"shift_mean"`
	ProportionalSkew float64         `yaml:"proportional_skew"`
	ClampMin         string          `yaml:"clamp_min"`
	ClampMax         string          `yaml:"clamp_max"`

	// CompiledTransform holds the pure math object once the YAML is parsed.
	// We ignore it during standard unmarshaling.
	CompiledTransform geom.Transform `yaml:"-"`
}

// DynamicDistribution wraps the pure math space with simulation-specific modifiers.
// This is the struct the rest of the simulation schema will use for all durations/probabilities.
type DynamicDistribution struct {
	Base      probability.SampleSpace `yaml:",inline"`
	Modifiers []DistributionModifier  `yaml:"modifiers,omitempty"`
}

// MarshalYAML resolves a known quirk in the go-yaml library where inlining
// a struct with a custom marshaler results in an empty object {}.
func (dd DynamicDistribution) MarshalYAML() (interface{}, error) {
	if len(dd.Modifiers) == 0 {
		return dd.Base.MarshalYAML()
	}

	typeStr := ""
	switch dd.Base.Type {
	case probability.ConstantDistribution:
		typeStr = "constant"
	case probability.NormalDistribution:
		typeStr = "normal"
	case probability.UniformDistribution:
		typeStr = "uniform"
	}

	type Alias struct {
		Type      string                  `yaml:"type"`
		Units     probability.SampleUnits `yaml:"units,omitempty"`
		Mean      float64                 `yaml:"mean,omitempty"`
		Min       float64                 `yaml:"min,omitempty"`
		Max       float64                 `yaml:"max,omitempty"`
		StdDev    float64                 `yaml:"std_dev,omitempty"`
		Const     float64                 `yaml:"value,omitempty"`
		Modifiers []DistributionModifier  `yaml:"modifiers,omitempty"`
	}

	return Alias{
		Type:      typeStr,
		Units:     dd.Base.Units,
		Mean:      dd.Base.Mean,
		Min:       dd.Base.Min,
		Max:       dd.Base.Max,
		StdDev:    dd.Base.StandardDeviation,
		Const:     dd.Base.Const,
		Modifiers: dd.Modifiers,
	}, nil
}

// UnmarshalYAML for DynamicDistribution intercepts the parsing to let the
// go-maths library handle the base space, while we compile the modifiers.
func (dd *DynamicDistribution) UnmarshalYAML(value *yaml.Node) error {
	// 1. Let the go-maths library parse the base values (e.g., mean: "15m", type: "normal")
	if err := value.Decode(&dd.Base); err != nil {
		return err
	}

	// 2. If it's just a scalar (e.g., duration: "15m"), we have no modifiers to parse.
	if value.Kind == yaml.ScalarNode {
		return nil
	}

	// 3. Otherwise, manually extract the modifiers array
	type Alias struct {
		Modifiers []DistributionModifier `yaml:"modifiers,omitempty"`
	}
	var a Alias
	if err := value.Decode(&a); err != nil {
		return err
	}

	// 4. Compile the YAML strings into pure geom.Transforms
	for i, mod := range a.Modifiers {
		transform := geom.Transform{
			ProportionalRate: mod.ProportionalSkew,
		}

		if mod.ShiftMean != "" {
			if d, err := time.ParseDuration(mod.ShiftMean); err == nil {
				transform.FlatShift = float64(d)
			}
		}
		if mod.ClampMin != "" {
			if d, err := time.ParseDuration(mod.ClampMin); err == nil {
				transform.HasMinClamp = true
				transform.ClampMin = float64(d)
			}
		}
		if mod.ClampMax != "" {
			if d, err := time.ParseDuration(mod.ClampMax); err == nil {
				transform.HasMaxClamp = true
				transform.ClampMax = float64(d)
			}
		}

		a.Modifiers[i].CompiledTransform = transform
	}

	dd.Modifiers = a.Modifiers
	return nil
}
