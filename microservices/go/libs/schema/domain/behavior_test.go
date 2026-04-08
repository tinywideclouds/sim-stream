// domain/behavior_test.go
package domain

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestWrapper is used to unmarshal the top-level YAML lists for testing.
type TestWrapper struct {
	RoutineTemplates []RoutineTemplate `yaml:"routine_templates"`
	Actors           []ActorTemplate   `yaml:"actors"`
	CollectiveEvents []CollectiveEvent `yaml:"collective_events"`
	Meters           []MeterTemplate   `yaml:"meters"`
	Actions          []ActionTemplate  `yaml:"actions"`
}

func TestRoutineBehaviorUnmarshaling(t *testing.T) {
	yamlInput := `
routine_templates:
  - routine_id: "morning_prep"
    description: "The ideal morning."
    tasks:
      - "morning_shower"
      - "make_toast"

actors:
  - actor_id: "parent_1"
    type: "adult"
    ai_model: "routine"
    routines:
      - routine_id: "morning_prep"
        trigger:
          type: 1 # Normal
          mean: "07h00m"
          std_dev: "5m"
        deadline:
          type: 1
          mean: "08h00m"
          std_dev: "2m"

collective_events:
  - event_id: "school_run"
    lead_actor: "child_1"
    dependent_actors: 
      - actor_id: "parent_1"
        friction_weight: 0.8
        patience_limit: "10m"
`

	var wrapper TestWrapper
	err := yaml.Unmarshal([]byte(strings.TrimSpace(yamlInput)), &wrapper)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	// 1. Verify Routine Templates
	if len(wrapper.RoutineTemplates) != 1 {
		t.Fatalf("Expected 1 routine template, got %d", len(wrapper.RoutineTemplates))
	}
	if wrapper.RoutineTemplates[0].Tasks[0] != "morning_shower" {
		t.Errorf("Expected first task to be 'morning_shower', got '%s'", wrapper.RoutineTemplates[0].Tasks[0])
	}

	// 2. Verify Actors and Fuzzy Deadlines
	if len(wrapper.Actors) != 1 {
		t.Fatalf("Expected 1 actor, got %d", len(wrapper.Actors))
	}

	actor := wrapper.Actors[0]
	if actor.AIModel != "routine" {
		t.Errorf("Expected AIModel 'routine', got '%s'", actor.AIModel)
	}

	deadline := actor.Routines[0].Deadline
	if deadline.Type != DistributionTypeNormal {
		t.Errorf("Expected deadline type to be %d, got %d", DistributionTypeNormal, deadline.Type)
	}

	// 3. Verify Collective Event Friction & Patience
	if len(wrapper.CollectiveEvents) != 1 {
		t.Fatalf("Expected 1 collective event, got %d", len(wrapper.CollectiveEvents))
	}
	event := wrapper.CollectiveEvents[0]
	if event.LeadActor != "child_1" {
		t.Errorf("Expected lead actor 'child_1', got '%s'", event.LeadActor)
	}
	if event.DependentActors[0].FrictionWeight != 0.8 {
		t.Errorf("Expected friction weight 0.8, got %f", event.DependentActors[0].FrictionWeight)
	}
}

func TestUtilityBehaviorUnmarshaling(t *testing.T) {
	yamlInput := `
meters:
  - meter_id: "hunger"
    max: 100.0
    base_decay_per_hour: 10.0
    curve: "exponential"

actions:
  - action_id: "cook_dinner"
    device_id: "cooker_1"
    satisfies:
      hunger: 80.0 # Proving backwards compatibility (flat float)
      energy:      # Proving new struct format
        amount: 15.0
        curve: "ease_in"
    costs:
      energy: 15.0
    duration:
      type: 3
      value: "45m"

actors:
  - actor_id: "wfh_worker"
    type: "adult"
    ai_model: "utility"
    starting_meters:
      hunger: 50.0
      duty: 0.0
`
	var wrapper TestWrapper
	err := yaml.Unmarshal([]byte(strings.TrimSpace(yamlInput)), &wrapper)
	if err != nil {
		t.Fatalf("Failed to unmarshal Utility YAML: %v", err)
	}

	if len(wrapper.Meters) != 1 || wrapper.Meters[0].BaseDecayPerHour != 10.0 {
		t.Errorf("Failed to parse Meters correctly")
	}

	// Verify ActionFill Unmarshaling
	if len(wrapper.Actions) != 1 {
		t.Fatalf("Expected 1 action, got %d", len(wrapper.Actions))
	}

	action := wrapper.Actions[0]
	if action.Satisfies["hunger"].Amount != 80.0 || action.Satisfies["hunger"].Curve != "linear" {
		t.Errorf("Failed to parse flat float backwards compatibility for 'hunger'")
	}
	if action.Satisfies["energy"].Amount != 15.0 || action.Satisfies["energy"].Curve != "ease_in" {
		t.Errorf("Failed to parse explicit struct for 'energy'")
	}

	actor := wrapper.Actors[0]
	if actor.AIModel != "utility" {
		t.Errorf("Expected AIModel 'utility', got '%s'", actor.AIModel)
	}
	if actor.StartingMeters["hunger"] != 50.0 {
		t.Errorf("Expected starting hunger 50.0, got %f", actor.StartingMeters["hunger"])
	}
}
