// internal/factory/ingest_test.go
package factory_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tinywideclouds/go-power-simulator/internal/factory"
)

func TestLoadRegistry_Success(t *testing.T) {
	// 1. Create a temporary mock catalog directory
	tempDir := t.TempDir()
	deviceDir := filepath.Join(tempDir, "devices")
	if err := os.Mkdir(deviceDir, 0755); err != nil {
		t.Fatalf("Failed to create mock device dir: %v", err)
	}

	// 2. Write a mock YAML file into the directory covering devices, actions, and the new schedules
	mockYAML := `
devices:
  - id: "budget_tv"
    tags: ["television"]
    template:
      device_id: "TBD"
      electrical_profile:
        max_watts: 50.0

actions:
  - id: "watch_tv"
    requires_device_tag: "television"
    template:
      action_id: "watch_tv"

routines:
  - id: "morning_prep"
    template:
      routine_id: "morning_prep"
      description: "Standard wake up sequence"
      tasks: ["use_bathroom", "make_coffee"]

collective_events:
  - id: "family_dinner"
    template:
      event_id: "family_dinner"
      lead_requirement: "adult" # UPDATED TO MATCH NEW SCHEMA
      action: "cook_family_meal"

compositions:
  - id: "test_recipe"
    description: "A test household"
    routine_ids: ["morning_prep"]
    event_ids: ["family_dinner"]
`
	filePath := filepath.Join(deviceDir, "mock_catalog.yaml")
	if err := os.WriteFile(filePath, []byte(mockYAML), 0644); err != nil {
		t.Fatalf("Failed to write mock yaml: %v", err)
	}

	// 3. Execute the Ingestion Engine
	reg, err := factory.LoadRegistry(tempDir)
	if err != nil {
		t.Fatalf("LoadRegistry failed: %v", err)
	}

	// 4. Verify Registry contents
	if len(reg.Devices) != 1 {
		t.Errorf("Expected 1 device, got %d", len(reg.Devices))
	}
	if dev, exists := reg.Devices["budget_tv"]; !exists || dev.Tags[0] != "television" {
		t.Errorf("Failed to properly parse and store the budget_tv device")
	}

	if len(reg.Actions) != 1 {
		t.Errorf("Expected 1 action, got %d", len(reg.Actions))
	}

	// Verify schedules were ingested
	if len(reg.Routines) != 1 {
		t.Errorf("Expected 1 routine, got %d", len(reg.Routines))
	}
	if rout, exists := reg.Routines["morning_prep"]; !exists || len(rout.Template.Tasks) != 2 {
		t.Errorf("Failed to properly parse and store the morning_prep routine")
	}

	if len(reg.CollectiveEvents) != 1 {
		t.Errorf("Expected 1 collective event, got %d", len(reg.CollectiveEvents))
	}
	// VERIFYING THE NEW FIELD
	if ev, exists := reg.CollectiveEvents["family_dinner"]; !exists || ev.Template.LeadRequirement != "adult" {
		t.Errorf("Failed to properly parse and store the family_dinner event")
	}

	// Verify composition references the schedule IDs
	if len(reg.Compositions) != 1 {
		t.Errorf("Expected 1 composition, got %d", len(reg.Compositions))
	}
	if comp, exists := reg.Compositions["test_recipe"]; !exists || comp.RoutineIDs[0] != "morning_prep" {
		t.Errorf("Failed to properly parse and store the composition recipe IDs")
	}
}

func TestLoadRegistry_DuplicateID_Fails(t *testing.T) {
	tempDir := t.TempDir()

	mockYAML := `
devices:
  - id: "duplicate_id"
  - id: "duplicate_id"
`
	filePath := filepath.Join(tempDir, "bad_data.yaml")
	_ = os.WriteFile(filePath, []byte(mockYAML), 0644)

	_, err := factory.LoadRegistry(tempDir)
	if err == nil {
		t.Errorf("Expected LoadRegistry to fail on duplicate ID, but it succeeded")
	}
}
