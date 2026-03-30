package domain

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestWrapper is used to unmarshal the top-level V2 YAML lists for testing.
type TestWrapper struct {
	RoutineTemplates []RoutineTemplate `yaml:"routine_templates"`
	Actors           []ActorTemplate   `yaml:"actors"`
	CollectiveEvents []CollectiveEvent `yaml:"collective_events"`
}

func TestV2BehaviorUnmarshaling(t *testing.T) {
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
    action: "leave_house"
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
	deadline := wrapper.Actors[0].Routines[0].Deadline
	if deadline.Type != DistributionTypeNormal {
		t.Errorf("Expected deadline type to be %d, got %d", DistributionTypeNormal, deadline.Type)
	}
	if deadline.Mean != "08h00m" {
		t.Errorf("Expected deadline mean to be '08h00m', got '%s'", deadline.Mean)
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
	if event.DependentActors[0].PatienceLimit != "10m" {
		t.Errorf("Expected patience limit '10m', got '%s'", event.DependentActors[0].PatienceLimit)
	}
}
