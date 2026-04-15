package domain

import (
	"testing"
	"time"

	"github.com/tinywideclouds/go-maths/pkg/probability"
	"gopkg.in/yaml.v3"
)

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
    blocks:
      - block_id: "core_sleep"
        probability: 1.0
        duration: "8h" # Testing the elegant scalar shorthand!
        modifiers:
          energy:
            amount: 75.0
            over: "8h"
            curve: "front_loaded"
  - phase_id: "work_trip"
    type: "away"
    anchor_time: "08:30"
    gravity: 0.9
    blocks:
      - block_id: "morning_commute"
        probability: 1.0
        duration:
          type: "normal"
          mean: "45m"
          std_dev: "10m"
          modifiers:
            - condition:
                context_key: "weather.is_raining"
                operator: "=="
                value: "true"
              shift_mean: "15m" # A 15 min delay if it's raining
        modifiers:
          energy:
            amount: -5.0
      - block_id: "office_shift"
        probability: 0.8
        duration:
          type: "normal"
          mean: "8h"
          std_dev: "30m"
        modifiers:
          hunger:
            amount: -40.0
            over: "8h"
`
	var actor testActorWrapper
	err := yaml.Unmarshal([]byte(yamlInput), &actor)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	if len(actor.Phases) != 2 {
		t.Fatalf("Expected 2 phases, got %d", len(actor.Phases))
	}

	// 1. Validate Sleep Phase (Testing scalar fallback logic)
	sleepPhase := actor.Phases[0]
	if sleepPhase.PhaseID != "night_sleep" || sleepPhase.Type != PhaseTypeSleep {
		t.Errorf("Sleep phase parsed incorrectly")
	}
	if len(sleepPhase.Blocks) != 1 {
		t.Fatalf("Expected 1 block in sleep phase")
	}

	sleepBlock := sleepPhase.Blocks[0]
	if sleepBlock.Duration.Base.Type != probability.ConstantDistribution {
		t.Errorf("Expected ConstantDistribution for scalar duration, got %v", sleepBlock.Duration.Base.Type)
	}
	if sleepBlock.Duration.Base.Const != float64(8*time.Hour) {
		t.Errorf("Expected const value %f, got %f", float64(8*time.Hour), sleepBlock.Duration.Base.Const)
	}

	sleepEnergy := sleepBlock.Modifiers["energy"]
	if sleepEnergy.Amount != 75.0 || sleepEnergy.Over != 8*time.Hour || sleepEnergy.Curve != "front_loaded" {
		t.Errorf("Sleep energy parsed incorrectly: %+v", sleepEnergy)
	}

	// 2. Validate Work Trip Phase (Testing full struct with math modifiers)
	workPhase := actor.Phases[1]
	if workPhase.Type != PhaseTypeAway {
		t.Errorf("Expected phase type 'away', got '%s'", workPhase.Type)
	}
	if len(workPhase.Blocks) != 2 {
		t.Fatalf("Expected 2 blocks in work phase")
	}

	commuteBlock := workPhase.Blocks[0]
	if commuteBlock.Duration.Base.Type != probability.NormalDistribution {
		t.Errorf("Expected NormalDistribution, got %v", commuteBlock.Duration.Base.Type)
	}
	if commuteBlock.Duration.Base.Mean != float64(45*time.Minute) {
		t.Errorf("Expected mean 45m, got %f", commuteBlock.Duration.Base.Mean)
	}

	// Check that the engine condition and compiled geom.Transform survived the parsing chain!
	if len(commuteBlock.Duration.Modifiers) != 1 {
		t.Fatalf("Expected 1 modifier on commute duration")
	}
	if commuteBlock.Duration.Modifiers[0].Condition.ContextKey != "weather.is_raining" {
		t.Errorf("Expected condition context key 'weather.is_raining'")
	}
	if commuteBlock.Duration.Modifiers[0].CompiledTransform.FlatShift != float64(15*time.Minute) {
		t.Errorf("Expected flat shift of 15m")
	}

	officeBlock := workPhase.Blocks[1]
	if officeBlock.Probability != 0.8 {
		t.Errorf("Expected 0.8 probability, got %f", officeBlock.Probability)
	}

	workHunger := officeBlock.Modifiers["hunger"]
	if workHunger.Amount != -40.0 {
		t.Errorf("Expected hunger -40.0, got %f", workHunger.Amount)
	}
}
