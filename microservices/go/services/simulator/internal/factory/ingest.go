// internal/factory/ingest.go
package factory

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// CatalogEnvelope represents the structure of a raw YAML file on disk.
type CatalogEnvelope struct {
	Devices      []CatalogDevice      `yaml:"devices"`
	Actions      []CatalogAction      `yaml:"actions"`
	Personas     []CatalogPersona     `yaml:"personas"`
	Systems      []CatalogSystem      `yaml:"systems"`
	Compositions []CatalogComposition `yaml:"compositions"`
	WaterSystems []CatalogWaterSystem `yaml:"water_systems"`

	Routines         []CatalogRoutine `yaml:"routines"`
	Alarms           []CatalogAlarm   `yaml:"alarms"`
	CollectiveEvents []CatalogEvent   `yaml:"collective_events"`
}

// LoadRegistry recursively walks a directory, parsing all YAML files into a new Registry.
func LoadRegistry(catalogDir string) (*Registry, error) {
	reg := NewRegistry()

	log.Printf("[DEBUG] Starting WalkDir in: %s", catalogDir)

	err := filepath.WalkDir(catalogDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || (!strings.HasSuffix(d.Name(), ".yaml") && !strings.HasSuffix(d.Name(), ".yml")) {
			return nil
		}

		log.Printf("[DEBUG] Found YAML file: %s", path)

		if err := IngestFile(path, reg); err != nil {
			return fmt.Errorf("failed to ingest %s: %w", path, err)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("catalog traversal failed: %w", err)
	}

	return reg, nil
}

// IngestFile reads a single YAML file and adds its contents to the Registry.
func IngestFile(filePath string, reg *Registry) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read file error: %w", err)
	}

	var envelope CatalogEnvelope
	if err := yaml.Unmarshal(data, &envelope); err != nil {
		return fmt.Errorf("yaml unmarshal error: %w", err)
	}

	log.Printf("[DEBUG] Parsed %s -> Devices: %d, Actions: %d, Personas: %d, Systems: %d, Compositions: %d",
		filePath, len(envelope.Devices), len(envelope.Actions), len(envelope.Personas), len(envelope.Systems), len(envelope.Compositions))

	for _, ws := range envelope.WaterSystems {
		if err := reg.AddWaterSystem(ws); err != nil {
			return err
		}
	}

	for i, dev := range envelope.Devices {
		log.Printf("[DEBUG] Registering Device [%d]: %s", i, dev.ID)
		if err := reg.AddDevice(dev); err != nil {
			return err
		}
	}

	for i, act := range envelope.Actions {
		log.Printf("[DEBUG] Registering Action [%d]: %s", i, act.ID)
		if err := reg.AddAction(act); err != nil {
			return err
		}
	}

	for i, per := range envelope.Personas {
		log.Printf("[DEBUG] Registering Persona [%d]: %s", i, per.ID)
		if err := reg.AddPersona(per); err != nil {
			return err
		}
	}

	for i, sys := range envelope.Systems {
		log.Printf("[DEBUG] Registering System [%d]: %s", i, sys.ID)
		if err := reg.AddSystem(sys); err != nil {
			return err
		}
	}

	for i, comp := range envelope.Compositions {
		log.Printf("[DEBUG] Registering Composition [%d]: %s", i, comp.ID)
		if err := reg.AddComposition(comp); err != nil {
			return err
		}
	}

	for _, rout := range envelope.Routines {
		if err := reg.AddRoutine(rout); err != nil {
			return err
		}
	}
	for _, alarm := range envelope.Alarms {
		if err := reg.AddAlarm(alarm); err != nil {
			return err
		}
	}
	for _, ev := range envelope.CollectiveEvents {
		if err := reg.AddEvent(ev); err != nil {
			return err
		}
	}

	return nil
}
