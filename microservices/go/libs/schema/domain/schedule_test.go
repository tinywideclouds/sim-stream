// domain/schedule_test.go
package domain

import (
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// testActorWrapper is used to isolate the test from changes to the full Actor struct
type testActorWrapper struct {
	ActorID string  `yaml:"actor_id"`
	Type    string  `yaml:"type"`
	AIModel string  `yaml:"ai_model"`
	Phases  []Phase `yaml:"phases"`
}

func TestPhaseUnmarshaling(t *testing.T) {
	yamlInput := `
actor_id: "office_worker"
type: "adult"
ai_model: "stable"
phases:
  - phase_id: "night_sleep"
    type: "sleep"
    anchor_time: "23:00"
    gravity: 0.2
    duration:
      type: 0 # Constant
      value: "8h"
      flexibility: "2h"
    modifiers:
      application: "continuous"
      effects:
        energy:
          amount: 75.0
          over: "8h"
          curve: "front_loaded"
  - phase_id: "work_shift"
    type: "away"
    anchor_time: "08:30"
    gravity: 0.9
    duration:
      type: 1 # Normal Distribution
      mean: "8h30m"
      std_dev: "30m"
      flexibility: "30m"
    modifiers:
      application: "block_end"
      effects:
        hunger:
          amount: -80.0
          over: "8h30m"
          curve: "linear"
`

	var actor testActorWrapper
	err := yaml.Unmarshal([]byte(strings.TrimSpace(yamlInput)), &actor)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	if len(actor.Phases) != 2 {
		t.Fatalf("Expected 2 phases, got %d", len(actor.Phases))
	}

	// 1. Validate the Elastic Sleep Phase
	sleepPhase := actor.Phases[0]
	if sleepPhase.Type != string(PhaseTypeSleep) {
		t.Errorf("Expected phase type 'sleep', got '%s'", sleepPhase.Type)
	}
	if sleepPhase.Gravity != 0.2 {
		t.Errorf("Expected gravity 0.2, got %f", sleepPhase.Gravity)
	}

	// Validate yaml.v3 parsed the human-readable flexibility string into a time.Duration
	if sleepPhase.Duration.Flexibility != 2*time.Hour {
		t.Errorf("Expected flexibility to parse as 2h, got %v", sleepPhase.Duration.Flexibility)
	}
	if sleepPhase.Modifiers.Application != string(Continuous) {
		t.Errorf("Expected continuous application, got %s", sleepPhase.Modifiers.Application)
	}

	// Validate the nested ContinuousEffect profile
	sleepEnergy, exists := sleepPhase.Modifiers.Effects["energy"]
	if !exists {
		t.Fatalf("Expected energy effect to be parsed")
	}
	if sleepEnergy.Amount != 75.0 {
		t.Errorf("Expected energy amount 75.0, got %f", sleepEnergy.Amount)
	}
	if sleepEnergy.Over != 8*time.Hour {
		t.Errorf("Expected energy over to parse as 8h, got %v", sleepEnergy.Over)
	}
	if sleepEnergy.Curve != "front_loaded" {
		t.Errorf("Expected front_loaded curve, got %s", sleepEnergy.Curve)
	}

	// 2. Validate the Strict Work Phase
	workPhase := actor.Phases[1]
	if workPhase.Type != string(PhaseTypeAway) {
		t.Errorf("Expected phase type 'away', got '%s'", workPhase.Type)
	}
	if workPhase.Duration.Flexibility != 30*time.Minute {
		t.Errorf("Expected flexibility to parse as 30m, got %v", workPhase.Duration.Flexibility)
	}
	if workPhase.Modifiers.Application != string(BlockEnd) {
		t.Errorf("Expected block_end application, got %s", workPhase.Modifiers.Application)
	}

	workHunger, exists := workPhase.Modifiers.Effects["hunger"]
	if !exists {
		t.Fatalf("Expected hunger effect to be parsed")
	}
	if workHunger.Amount != -80.0 {
		t.Errorf("Expected hunger modifier -80.0, got %f", workHunger.Amount)
	}

	// Complex duration check (8h30m)
	expectedWorkDur := 8*time.Hour + 30*time.Minute
	if workHunger.Over != expectedWorkDur {
		t.Errorf("Expected hunger over to parse as 8h30m, got %v", workHunger.Over)
	}
}
