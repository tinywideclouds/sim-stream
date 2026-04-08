// domain/schedule_test.go
package domain

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestPhaseUnmarshaling(t *testing.T) {
	yamlInput := `
actor_id: "office_worker"
type: "adult"
ai_model: "stable"
phases:
  - phase_id: "morning_prep"
    anchor_time: "07:00"
    buffer_duration: "30m"
    type: "home"
  - phase_id: "work_shift"
    anchor_time: "08:30"
    buffer_duration: "15m"
    type: "away"
    away_profile:
      duration:
        type: 1 # Normal Distribution
        mean: "8h30m"
        std_dev: "30m"
      modifiers:
        energy: -45.0
        hunger: -80.0
`

	var actor ActorTemplate
	err := yaml.Unmarshal([]byte(strings.TrimSpace(yamlInput)), &actor)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	if len(actor.Phases) != 2 {
		t.Fatalf("Expected 2 phases, got %d", len(actor.Phases))
	}

	workPhase := actor.Phases[1]
	if workPhase.Type != PhaseTypeAway {
		t.Errorf("Expected phase type 'away', got '%s'", workPhase.Type)
	}

	if workPhase.AwayProfile == nil {
		t.Fatal("Expected AwayProfile to be populated")
	}

	if workPhase.AwayProfile.Modifiers["hunger"] != -80.0 {
		t.Errorf("Expected hunger modifier -80.0, got %f", workPhase.AwayProfile.Modifiers["hunger"])
	}
}
