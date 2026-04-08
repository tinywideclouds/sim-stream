// internal/factory/registry_test.go
package factory_test

import (
	"testing"

	"github.com/tinywideclouds/go-power-simulator/internal/factory"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

func TestRegistry_AddComponents(t *testing.T) {
	reg := factory.NewRegistry()

	dev := factory.CatalogDevice{
		ID:   "standard_tv",
		Tags: []string{"television", "entertainment"},
		Template: domain.DeviceTemplate{
			DeviceID: "WILL_BE_OVERWRITTEN_BY_GENERATOR",
		},
	}

	act := factory.CatalogAction{
		ID:                "watch_tv",
		RequiresDeviceTag: "television",
		Template: domain.ActionTemplate{
			ActionID: "watch_tv",
		},
	}

	if err := reg.AddDevice(dev); err != nil {
		t.Fatalf("Failed to add device: %v", err)
	}
	if err := reg.AddAction(act); err != nil {
		t.Fatalf("Failed to add action: %v", err)
	}

	// Verify collision detection
	if err := reg.AddDevice(dev); err == nil {
		t.Errorf("Expected error when adding duplicate device ID")
	}

	if len(reg.Devices) != 1 || len(reg.Actions) != 1 {
		t.Errorf("Registry did not store components correctly")
	}
}
