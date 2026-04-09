package main

import (
	"flag"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/tinywideclouds/go-power-simulator/internal/factory"
	"github.com/tinywideclouds/go-sim-probability/pkg/generator"
	"gopkg.in/yaml.v3"
)

func main() {
	catalogsDir := flag.String("catalogs", "./catalogs", "Path to the modular YAML catalogs")
	outDir := flag.String("out", "./setup/bulk", "Output directory for generated households")
	totalRuns := flag.Int("count", 100, "Number of households to generate")
	flag.Parse()

	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, opts))
	slog.SetDefault(logger)

	slog.Info("--- Starting Bulk Household Profiler ---")

	// 1. Load the Registry
	reg, err := factory.LoadRegistry(*catalogsDir)
	if err != nil {
		slog.Error("Critical: Failed to load catalog registry", slog.Any("error", err))
		os.Exit(1)
	}

	if len(reg.Compositions) == 0 {
		slog.Error("Critical: No household compositions found", slog.String("directory", *catalogsDir))
		os.Exit(1)
	}

	slog.Info("Registry loaded",
		slog.Int("devices", len(reg.Devices)),
		slog.Int("actions", len(reg.Actions)),
		slog.Int("personas", len(reg.Personas)),
		slog.Int("recipes", len(reg.Compositions)))

	// 2. Prepare Output Environment
	if err := os.MkdirAll(*outDir, 0755); err != nil {
		slog.Error("Failed to create output directory", slog.Any("error", err))
		os.Exit(1)
	}

	// 3. Initialize the Monte Carlo Pipeline
	var seed [32]byte
	copy(seed[:], []byte(time.Now().String()))
	sampler := generator.NewSampler(seed)
	builder := factory.NewHouseholdGenerator(reg, sampler)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Convert Compositions map to a slice for random selection
	var recipes []factory.CatalogComposition
	for _, comp := range reg.Compositions {
		recipes = append(recipes, comp)
	}

	// Index Personas by their Type (adult, child, elderly)
	personasByType := make(map[string][]string)
	for id, p := range reg.Personas {
		personasByType[p.Type] = append(personasByType[p.Type], id)
	}

	// 4. The Generation Loop
	successCount := 0
	for i := 1; i <= *totalRuns; i++ {
		recipe := recipes[rng.Intn(len(recipes))]

		req := factory.GenerationRequest{
			ArchetypeID:            fmt.Sprintf("%s_%03d", recipe.ID, i),
			RequiredDeviceTags:     recipe.RequiredDeviceTags,
			RequiredWaterSystemTag: recipe.RequiredWaterSystemTag,
			SystemIDs:              recipe.SystemIDs,
			RoutineIDs:             recipe.RoutineIDs,
			AlarmIDs:               recipe.AlarmIDs,
			EventIDs:               recipe.EventIDs,
		}

		// RESOLVE DYNAMIC PERSONA REQUIREMENTS
		for _, preq := range recipe.PersonaRequirements {
			available := personasByType[preq.Type]
			if len(available) == 0 {
				if preq.Min > 0 {
					slog.Warn("Recipe requires persona type but none exist", slog.String("recipe", recipe.ID), slog.String("type", preq.Type))
				}
				continue
			}

			// Roll how many of this type we need
			personaCount := preq.Min
			if preq.Max > preq.Min {
				personaCount += rng.Intn(preq.Max - preq.Min + 1)
			}

			// Randomly select them from the available bucket
			for c := 0; c < personaCount; c++ {
				pID := available[rng.Intn(len(available))]
				req.PersonaIDs = append(req.PersonaIDs, pID)
			}
		}

		// Execute the Assembly
		node, err := builder.Generate(req)
		if err != nil {
			slog.Warn("Skipping house", slog.Int("index", i), slog.String("recipe", recipe.ID), slog.Any("error", err))
			continue
		}

		// Serialize & Write
		outData, err := yaml.Marshal(node)
		if err != nil {
			slog.Error("Failed to marshal YAML", slog.String("archetype", req.ArchetypeID), slog.Any("error", err))
			os.Exit(1)
		}

		fileName := fmt.Sprintf("%s.yaml", req.ArchetypeID)
		filePath := filepath.Join(*outDir, fileName)

		if err := os.WriteFile(filePath, outData, 0644); err != nil {
			slog.Error("Failed to write file", slog.String("path", filePath), slog.Any("error", err))
			os.Exit(1)
		}

		successCount++
	}

	slog.Info("--- Profiler Finished ---")
	slog.Info("Successfully generated unique households", slog.Int("count", successCount), slog.String("directory", *outDir))
}
