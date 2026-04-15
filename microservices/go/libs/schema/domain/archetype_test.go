package domain

import (
	"strings"
	"testing"

	"github.com/tinywideclouds/go-maths/pkg/probability"
	"gopkg.in/yaml.v3"
)

func TestNodeArchetypeUnmarshaling(t *testing.T) {
	yamlInput := `
archetype_id: "test_house_01"
description: "A complete multi-agent integrated configuration test."
base_temp_c: 12.0
insulation_decay_rate: 0.15

devices:
  - device_id: "shower_1"
    taxonomy:
      category: 5 # Wet Appliance
      class_name: "electric_shower"
    electrical_profile:
      type: 1
      max_watts: 0.0 # Boiler handles the heat
    water_profile:
      hot_lpm: 9.5
      cold_lpm: 0.0

scenarios:
  - scenario_id: "thermostat"
    actor_tags: ["system"]
    trigger:
      type: 2
      base_conditions:
        - context_key: "indoor_temp_c"
          operator: 4
          value: "16.5"
      fatigue_rule:
        lockout_duration: "30m" # Beautiful new scalar shorthand
    actions:
      - device_id: "central_heating"
        state: 1
        parameters:
          duration:
            type: "uniform" # Updated to new go-maths string enum
            min: "1.0"      # Updated to flexible string parsing
            max: "2.5"
`

	var archetype NodeArchetype
	err := yaml.Unmarshal([]byte(strings.TrimSpace(yamlInput)), &archetype)
	if err != nil {
		t.Fatalf("Failed to unmarshal archetype YAML: %v", err)
	}

	// 1. Root Properties
	if archetype.ArchetypeID != "test_house_01" {
		t.Errorf("Expected ArchetypeID 'test_house_01', got '%s'", archetype.ArchetypeID)
	}
	if archetype.BaseTempC != 12.0 {
		t.Errorf("Expected BaseTempC 12.0, got %f", archetype.BaseTempC)
	}

	// 2. Device Unmarshaling & Water Profile Pointers
	if len(archetype.Devices) != 1 {
		t.Fatalf("Expected 1 device, got %d", len(archetype.Devices))
	}
	if archetype.Devices[0].WaterProfile == nil {
		t.Fatal("Expected WaterProfile to be populated, got nil")
	}
	if archetype.Devices[0].WaterProfile.HotLitersPerMinute != 9.5 {
		t.Errorf("Expected hot_lpm 9.5, got %f", archetype.Devices[0].WaterProfile.HotLitersPerMinute)
	}

	// 3. Scenario & Trigger Rules
	if len(archetype.Scenarios) != 1 {
		t.Fatalf("Expected 1 scenario, got %d", len(archetype.Scenarios))
	}

	trigger := archetype.Scenarios[0].Trigger
	if trigger.Type != TriggerTypeEventReaction {
		t.Errorf("Expected TriggerType %d, got %d", TriggerTypeEventReaction, trigger.Type)
	}
	if len(trigger.BaseConditions) != 1 || trigger.BaseConditions[0].ContextKey != "indoor_temp_c" {
		t.Errorf("Expected BaseCondition on indoor_temp_c")
	}

	// Ensure the pure math parser picked up the lockout duration
	if trigger.FatigueRule.LockoutDuration.Type != probability.ConstantDistribution {
		t.Errorf("Expected Constant distribution for lockout, got %v", trigger.FatigueRule.LockoutDuration.Type)
	}

	actionParam := archetype.Scenarios[0].Actions[0].Parameters["duration"]
	if actionParam.Type != probability.UniformDistribution {
		t.Errorf("Expected Uniform distribution in action parameter, got %v", actionParam.Type)
	}
	if actionParam.Min != 1.0 || actionParam.Max != 2.5 {
		t.Errorf("Expected parameter bounds 1.0 to 2.5, got %f to %f", actionParam.Min, actionParam.Max)
	}
}
