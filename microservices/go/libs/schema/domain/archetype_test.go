package domain

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestNodeArchetypeUnmarshaling(t *testing.T) {
	yamlInput := `
archetype_id: "v2_test_house_01"
description: "A complete multi-agent V2 configuration test."
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
        lockout_duration: "30m"
    actions:
      - device_id: "central_heating"
        state: 1
        parameters:
          duration:
            type: 2
            min: 1.0
            max: 2.5
`

	var archetype NodeArchetype
	err := yaml.Unmarshal([]byte(strings.TrimSpace(yamlInput)), &archetype)
	if err != nil {
		t.Fatalf("Failed to unmarshal archetype YAML: %v", err)
	}

	// 1. Root Properties
	if archetype.ArchetypeID != "v2_test_house_01" {
		t.Errorf("Expected ArchetypeID 'v2_test_house_01', got '%s'", archetype.ArchetypeID)
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

	// 3. Scenario & New Trigger Rules
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
	if trigger.FatigueRule.LockoutDuration != "30m" {
		t.Errorf("Expected FatigueRule lockout to be '30m', got '%s'", trigger.FatigueRule.LockoutDuration)
	}
}
