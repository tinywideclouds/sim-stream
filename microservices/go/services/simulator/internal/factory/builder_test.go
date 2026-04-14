// internal/factory/builder_test.go
package factory

import (
	"testing"

	"github.com/tinywideclouds/go-sim-probability/pkg/generator"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

func TestHouseholdGenerator_Generate_Schedules(t *testing.T) {
	reg := NewRegistry()

	_ = reg.AddPersona(CatalogPersona{
		ID:        "adult_test",
		Type:      "adult",
		Frequency: 100,
		StartingMeters: map[string]domain.ProbabilityDistribution{
			"energy": {Type: domain.DistributionTypeConstant, Value: "80.0"},
		},
		Biology: map[string]domain.BiologyConfig{
			"hunger": {
				DecayPerHour:     domain.ProbabilityDistribution{Type: domain.DistributionTypeConstant, Value: "5.5"},
				PhaseMultipliers: map[string]float64{"sleep": 0.1},
			},
		},
	})

	_ = reg.AddRoutine(CatalogRoutine{
		ID:       "morning_prep",
		Template: domain.RoutineTemplate{RoutineID: "morning_prep"},
	})

	_ = reg.AddAlarm(CatalogAlarm{
		ID:       "wakeup_alarm",
		Template: domain.AlarmTemplate{AlarmID: "wakeup_alarm"},
	})

	_ = reg.AddEvent(CatalogEvent{
		ID:        "family_dinner",
		Selection: SelectionBlock{Weight: 1.0},
		Template: CatalogEventTemplate{
			EventID:         "family_dinner",
			Action:          "cook_large_dinner",
			LeadRequirement: "adult",
		},
	})

	sampler := generator.NewSampler([32]byte{})
	builder := NewHouseholdGenerator(reg, sampler)

	req := GenerationRequest{
		ArchetypeID: "test_household",
		// Replace explicit PersonaIDs with requirements bounding Min/Max counts
		PersonaRequirements: []PersonaRequirement{
			{Type: "adult", Min: 1, Max: 1},
		},
		RoutineIDs: []string{"morning_prep"},
		AlarmIDs:   []string{"wakeup_alarm"},
		EventIDs:   []string{"family_dinner"},
	}

	node, err := builder.Generate(req)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if len(node.Actors) != 1 {
		t.Fatalf("Expected exactly 1 actor, got %d", len(node.Actors))
	}

	// Verify the Instantiated Gaussian values
	actor := node.Actors[0]
	if actor.StartingMeters["energy"] != 80.0 {
		t.Errorf("Expected starting energy 80.0, got %f", actor.StartingMeters["energy"])
	}
	if bio, ok := actor.Biology["hunger"]; !ok || bio.DecayPerHour != 5.5 {
		t.Errorf("Expected biological hunger decay 5.5, got %v", bio)
	}

	if len(node.RoutineTemplates) != 1 || node.RoutineTemplates[0].RoutineID != "morning_prep" {
		t.Errorf("Expected exactly 1 routine 'morning_prep', got %v", node.RoutineTemplates)
	}

	if len(node.Alarms) != 1 || node.Alarms[0].AlarmID != "wakeup_alarm" {
		t.Errorf("Expected exactly 1 alarm 'wakeup_alarm', got %v", node.Alarms)
	}

	if len(node.CollectiveEvents) != 1 || node.CollectiveEvents[0].EventID != "family_dinner" {
		t.Errorf("Expected exactly 1 collective event 'family_dinner', got %v", node.CollectiveEvents)
	}
}

func TestHouseholdGenerator_Generate_MissingSchedule(t *testing.T) {
	reg := NewRegistry()
	sampler := generator.NewSampler([32]byte{})
	builder := NewHouseholdGenerator(reg, sampler)

	req := GenerationRequest{
		ArchetypeID: "test_household",
		RoutineIDs:  []string{"non_existent_routine"},
	}

	_, err := builder.Generate(req)
	if err == nil {
		t.Fatal("Expected error when requesting missing routine, got nil")
	}
}
