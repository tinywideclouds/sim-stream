package domain

import (
	"strings"
	"testing"
	"time"

	"github.com/tinywideclouds/go-maths/pkg/probability"
	"gopkg.in/yaml.v3"
)

type RoutineTestWrapper struct {
	Actors []testActor `yaml:"actors"`
}

type testActor struct {
	ActorID  string         `yaml:"actor_id"`
	Routines []ActorRoutine `yaml:"routines"`
}

func TestActorRoutine_DynamicDistributionParsing(t *testing.T) {
	yamlInput := `
actors:
  - actor_id: "office_worker"
    routines:
      - routine_id: "morning_prep"
        trigger:
          type: "normal"
          mean: "07h00m"
          std_dev: "5m"
        deadline: "08h30m" # Scalar shorthand test!
`
	var wrapper RoutineTestWrapper
	err := yaml.Unmarshal([]byte(strings.TrimSpace(yamlInput)), &wrapper)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	routine := wrapper.Actors[0].Routines[0]

	// 1. Verify Full Struct parsing
	if routine.Trigger.Base.Type != probability.NormalDistribution {
		t.Errorf("Expected Normal distribution for trigger, got %v", routine.Trigger.Base.Type)
	}
	expectedMean := float64(7 * time.Hour)
	if routine.Trigger.Base.Mean != expectedMean {
		t.Errorf("Expected mean %f, got %f", expectedMean, routine.Trigger.Base.Mean)
	}

	// 2. Verify Scalar Shorthand parsing
	if routine.Deadline.Base.Type != probability.ConstantDistribution {
		t.Errorf("Expected Constant distribution for deadline shorthand, got %v", routine.Deadline.Base.Type)
	}
	expectedDeadline := float64(8*time.Hour + 30*time.Minute)
	if routine.Deadline.Base.Const != expectedDeadline {
		t.Errorf("Expected deadline const %f, got %f", expectedDeadline, routine.Deadline.Base.Const)
	}
}
