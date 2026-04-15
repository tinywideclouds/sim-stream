package domain

import (
	"strings"
	"testing"
	"time"

	"github.com/tinywideclouds/go-maths/pkg/probability"
	"gopkg.in/yaml.v3"
)

// TestWrapper isolates the DynamicDistribution unmarshaling for pure testing.
type TestWrapper struct {
	Duration DynamicDistribution `yaml:"duration"`
}

func TestDynamicDistribution_UnmarshalYAML_ScalarFallback(t *testing.T) {
	yamlInput := `
duration: "45m"
`
	var wrapper TestWrapper
	err := yaml.Unmarshal([]byte(strings.TrimSpace(yamlInput)), &wrapper)
	if err != nil {
		t.Fatalf("Failed to unmarshal scalar YAML: %v", err)
	}

	// 1. Verify it delegated to go-maths and set the Constant fallback
	if wrapper.Duration.Base.Type != probability.ConstantDistribution {
		t.Errorf("Expected ConstantDistribution (0), got %v", wrapper.Duration.Base.Type)
	}
	expectedNs := float64(45 * time.Minute)
	if wrapper.Duration.Base.Const != expectedNs {
		t.Errorf("Expected const value %f, got %f", expectedNs, wrapper.Duration.Base.Const)
	}
}

func TestDynamicDistribution_UnmarshalYAML_FullStruct(t *testing.T) {
	yamlInput := `
duration:
  type: "normal"
  mean: "8h"
  std_dev: "30m"
  modifiers:
    - condition:
        context_key: "weather.external_temp_c"
        operator: "<"
        value: "5.0"
      shift_mean: "30m"
      proportional_skew: 2.0
      clamp_max: "45m"
`
	var wrapper TestWrapper
	err := yaml.Unmarshal([]byte(strings.TrimSpace(yamlInput)), &wrapper)
	if err != nil {
		t.Fatalf("Failed to unmarshal full YAML: %v", err)
	}

	// 1. Verify Base go-maths struct
	if wrapper.Duration.Base.Type != probability.NormalDistribution {
		t.Errorf("Expected NormalDistribution, got %v", wrapper.Duration.Base.Type)
	}
	expectedMeanNs := float64(8 * time.Hour)
	if wrapper.Duration.Base.Mean != expectedMeanNs {
		t.Errorf("Expected mean to compile to %f, got %f", expectedMeanNs, wrapper.Duration.Base.Mean)
	}

	// 2. Verify Modifiers Array Length
	if len(wrapper.Duration.Modifiers) != 1 {
		t.Fatalf("Expected 1 modifier, got %d", len(wrapper.Duration.Modifiers))
	}

	mod := wrapper.Duration.Modifiers[0]

	// 3. Verify Condition String Parsing
	if mod.Condition.Operator != ConditionOperatorLt {
		t.Errorf("Expected strictly less-than operator, got %v", mod.Condition.Operator)
	}

	// 4. Verify the compiled geom.Transform math block!
	if mod.CompiledTransform.FlatShift != float64(30*time.Minute) {
		t.Errorf("Expected FlatShift of 30m (%f), got %f", float64(30*time.Minute), mod.CompiledTransform.FlatShift)
	}
	if mod.CompiledTransform.ProportionalRate != 2.0 {
		t.Errorf("Expected ProportionalRate of 2.0, got %f", mod.CompiledTransform.ProportionalRate)
	}
	if !mod.CompiledTransform.HasMaxClamp {
		t.Errorf("Expected HasMaxClamp to be true")
	}
	if mod.CompiledTransform.ClampMax != float64(45*time.Minute) {
		t.Errorf("Expected ClampMax of 45m (%f), got %f", float64(45*time.Minute), mod.CompiledTransform.ClampMax)
	}
}
