// internal/factory/ingest.go
package factory

import (
	"fmt"
	"io/fs"
	"log/slog"
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

	slog.Debug("Starting WalkDir", slog.String("directory", catalogDir))

	err := filepath.WalkDir(catalogDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || (!strings.HasSuffix(d.Name(), ".yaml") && !strings.HasSuffix(d.Name(), ".yml")) {
			return nil
		}

		slog.Debug("Found YAML file", slog.String("file", path))

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

	slog.Info("Parsed YAML file",
		slog.String("file", filePath),
		slog.Int("devices", len(envelope.Devices)),
		slog.Int("actions", len(envelope.Actions)),
		slog.Int("personas", len(envelope.Personas)),
		slog.Int("systems", len(envelope.Systems)),
		slog.Int("compositions", len(envelope.Compositions)),
		slog.Int("events", len(envelope.CollectiveEvents)),
	)

	for _, ws := range envelope.WaterSystems {
		if err := reg.AddWaterSystem(ws); err != nil {
			return err
		}
	}

	for i, dev := range envelope.Devices {
		slog.Debug("Registering Device", slog.Int("index", i), slog.String("id", dev.ID))
		if err := reg.AddDevice(dev); err != nil {
			return err
		}
	}

	for i, act := range envelope.Actions {
		slog.Debug("Registering Action", slog.Int("index", i), slog.String("id", act.ID))
		if err := reg.AddAction(act); err != nil {
			return err
		}
	}

	for i, per := range envelope.Personas {
		slog.Debug("Registering Persona", slog.Int("index", i), slog.String("id", per.ID))
		if err := reg.AddPersona(per); err != nil {
			return err
		}
	}

	for i, sys := range envelope.Systems {
		slog.Debug("Registering System", slog.Int("index", i), slog.String("id", sys.ID))
		if err := reg.AddSystem(sys); err != nil {
			return err
		}
	}

	for i, comp := range envelope.Compositions {
		slog.Debug("Registering Composition", slog.Int("index", i), slog.String("id", comp.ID))
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
