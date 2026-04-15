package probability

import (
	"fmt"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

type ProbabilityDistribution int

const (
	// By making Constant 0, an uninitialized SampleSpace automatically acts as a mathematical constant of 0.0.
	// It also allows users to omit 'type: constant' from the YAML entirely for shorthand values.
	ConstantDistribution ProbabilityDistribution = iota
	NormalDistribution
	UniformDistribution
)

func (pd *ProbabilityDistribution) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}

	switch s {
	case "normal":
		*pd = NormalDistribution
	case "uniform":
		*pd = UniformDistribution
	case "constant":
		*pd = ConstantDistribution
	default:
		// Fail fast if they typed 'nrmal' instead of silently defaulting
		return fmt.Errorf("unknown distribution type: %q", s)
	}
	return nil
}

// SampleUnits allows us to define what the floats represent.
// If a time metric is required, Duration is populated so we can cast back cleanly.
type SampleUnits struct {
	Metric   string
	Duration *time.Duration
}

type SampleSpace struct {
	Type              ProbabilityDistribution `yaml:"type"`
	Units             SampleUnits             `yaml:"units,omitempty"`
	Mean              float64                 `yaml:"-"`
	Min               float64                 `yaml:"-"`
	Max               float64                 `yaml:"-"`
	StandardDeviation float64                 `yaml:"-"`
	Const             float64                 `yaml:"-"`
}

func (ss SampleSpace) MarshalYAML() (interface{}, error) {
	// For simple constants without units, just return the raw float
	if ss.Type == ConstantDistribution && ss.Units.Duration == nil {
		return ss.Const, nil
	}

	typeStr := ""
	switch ss.Type {
	case ConstantDistribution:
		typeStr = "constant"
	case NormalDistribution:
		typeStr = "normal"
	case UniformDistribution:
		typeStr = "uniform"
	}

	type Alias struct {
		Type   string      `yaml:"type"`
		Units  SampleUnits `yaml:"units,omitempty"`
		Mean   float64     `yaml:"mean,omitempty"`
		Min    float64     `yaml:"min,omitempty"`
		Max    float64     `yaml:"max,omitempty"`
		StdDev float64     `yaml:"std_dev,omitempty"`
		Const  float64     `yaml:"value,omitempty"`
	}

	return Alias{
		Type:   typeStr,
		Units:  ss.Units,
		Mean:   ss.Mean,
		Min:    ss.Min,
		Max:    ss.Max,
		StdDev: ss.StandardDeviation,
		Const:  ss.Const,
	}, nil
}

// UnmarshalYAML compiles human-readable strings ("15m" or "42.5")
// into pure float64 math spaces exactly once during startup.
func (ss *SampleSpace) UnmarshalYAML(value *yaml.Node) error {
	// If the user just wrote a raw string like `duration: "15m"`, we handle it instantly
	// Because Type defaults to 0 (ConstantDistribution), this works perfectly.
	if value.Kind == yaml.ScalarNode {
		val, err := parseFlexibleValue(value.Value)
		if err != nil {
			return err
		}
		ss.Const = val
		return nil
	}

	// Alias prevents infinite recursion while allowing us to grab raw strings from a struct block
	type Alias struct {
		Type   ProbabilityDistribution `yaml:"type"`
		Units  SampleUnits             `yaml:"units,omitempty"`
		Mean   string                  `yaml:"mean,omitempty"`
		Min    string                  `yaml:"min,omitempty"`
		Max    string                  `yaml:"max,omitempty"`
		StdDev string                  `yaml:"std_dev,omitempty"`
		Const  string                  `yaml:"value,omitempty"` // Fallback to "value" keyword for constants
	}

	var a Alias
	if err := value.Decode(&a); err != nil {
		return err
	}

	ss.Type = a.Type
	ss.Units = a.Units

	var err error
	if ss.Mean, err = parseFlexibleValue(a.Mean); err != nil {
		return err
	}
	if ss.Min, err = parseFlexibleValue(a.Min); err != nil {
		return err
	}
	if ss.Max, err = parseFlexibleValue(a.Max); err != nil {
		return err
	}
	if ss.StandardDeviation, err = parseFlexibleValue(a.StdDev); err != nil {
		return err
	}
	if ss.Const, err = parseFlexibleValue(a.Const); err != nil {
		return err
	}

	return nil
}

// parseFlexibleValue attempts to parse as a duration first (converting to ns),
// then falls back to a standard float64.
func parseFlexibleValue(input string) (float64, error) {
	if input == "" {
		return 0, nil
	}
	if d, err := time.ParseDuration(input); err == nil {
		return float64(d), nil
	}
	return strconv.ParseFloat(input, 64)
}
