// internal/reporting/csv_test.go
package reporting_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tinywideclouds/go-power-simulator/internal/reporting"
)

func TestCSVReporter_WritesCorrectly(t *testing.T) {
	// Create a safe temporary directory for file I/O testing
	tempDir := t.TempDir()

	reporter, err := reporting.NewCSVReporter(tempDir)
	if err != nil {
		t.Fatalf("Failed to create reporter: %v", err)
	}

	simTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	// Test Actor Writing (With Shared Context)
	err = reporter.AddActorAction("house_1", "actor_1", "cook_lunch", true, simTime)
	if err != nil {
		t.Errorf("AddActorAction failed: %v", err)
	}

	// Test Power Writing
	err = reporter.AddPowerUsage("house_1", simTime, 2500.5, 21.0, 22.0, 55.0, []string{"oven_1", "tv_1"})
	if err != nil {
		t.Errorf("AddPowerUsage failed: %v", err)
	}

	// Test Meter Writing
	err = reporter.AddActorMeters("house_1", "actor_1", simTime, 45.5, 20.0, 90.0, 15.2)
	if err != nil {
		t.Errorf("AddActorMeters failed: %v", err)
	}

	// Close to flush buffers to disk
	reporter.Close()

	// Verify Actor File Content
	actorData, err := os.ReadFile(filepath.Join(tempDir, "_actor_timeline.csv"))
	if err != nil {
		t.Fatalf("Failed to read actor file: %v", err)
	}
	actorLines := strings.Split(strings.TrimSpace(string(actorData)), "\n")

	expectedActorHeader := "HouseholdID,Timestamp,ActorID,ActionID,IsShared"
	if actorLines[0] != expectedActorHeader {
		t.Errorf("Expected actor header %q, got %q", expectedActorHeader, actorLines[0])
	}
	expectedActorRow := "house_1,2026-01-01T12:00:00Z,actor_1,cook_lunch,true"
	if actorLines[1] != expectedActorRow {
		t.Errorf("Expected actor row %q, got %q", expectedActorRow, actorLines[1])
	}

	// Verify Power File Content
	powerData, err := os.ReadFile(filepath.Join(tempDir, "_power_usage.csv"))
	if err != nil {
		t.Fatalf("Failed to read power file: %v", err)
	}
	powerLines := strings.Split(strings.TrimSpace(string(powerData)), "\n")

	expectedPowerHeader := "HouseholdID,Timestamp,TotalWatts,IndoorTempC,TankTempC,ActiveDevices"
	if powerLines[0] != expectedPowerHeader {
		t.Errorf("Expected power header %q, got %q", expectedPowerHeader, powerLines[0])
	}
	expectedPowerRow := "house_1,2026-01-01T12:00:00Z,2500.50,21.00,55.00,oven_1|tv_1"
	if powerLines[1] != expectedPowerRow {
		t.Errorf("Expected power row %q, got %q", expectedPowerRow, powerLines[1])
	}

	// Verify Meter File Content
	meterData, err := os.ReadFile(filepath.Join(tempDir, "_actor_meters.csv"))
	if err != nil {
		t.Fatalf("Failed to read meter file: %v", err)
	}
	meterLines := strings.Split(strings.TrimSpace(string(meterData)), "\n")

	expectedMeterHeader := "HouseholdID,Timestamp,ActorID,Energy,Hunger,Hygiene,Leisure"
	if meterLines[0] != expectedMeterHeader {
		t.Errorf("Expected meter header %q, got %q", expectedMeterHeader, meterLines[0])
	}
	expectedMeterRow := "house_1,2026-01-01T12:00:00Z,actor_1,45.50,20.00,90.00,15.20"
	if meterLines[1] != expectedMeterRow {
		t.Errorf("Expected meter row %q, got %q", expectedMeterRow, meterLines[1])
	}
}
