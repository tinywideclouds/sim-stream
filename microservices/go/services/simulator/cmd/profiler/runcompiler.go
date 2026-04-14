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

	opts := &slog.HandlerOptions{Level: slog.LevelDebug}
	logger := slog.New(slog.NewTextHandler(os.Stdout, opts))
	slog.SetDefault(logger)

	slog.Info("--- Starting Bulk Household Profiler ---")

	reg, err := factory.LoadRegistry(*catalogsDir)
	if err != nil {
		slog.Error("Critical: Failed to load catalog registry", slog.Any("error", err))
		os.Exit(1)
	}

	if len(reg.Compositions) == 0 {
		slog.Error("Critical: No household compositions found", slog.String("directory", *catalogsDir))
		os.Exit(1)
	}

	if err := os.MkdirAll(*outDir, 0755); err != nil {
		slog.Error("Failed to create output directory", slog.Any("error", err))
		os.Exit(1)
	}

	var seed [32]byte
	copy(seed[:], []byte(time.Now().String()))
	sampler := generator.NewSampler(seed)
	builder := factory.NewHouseholdGenerator(reg, sampler)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	var recipes []factory.CatalogComposition
	totalRecipeWeight := 0
	for _, comp := range reg.Compositions {
		recipes = append(recipes, comp)
		w := comp.Frequency
		if w <= 0 {
			w = 10
		}
		totalRecipeWeight += w
	}

	successCount := 0
	for i := 1; i <= *totalRuns; i++ {
		// 1. Weighted Recipe Selection
		roll := rng.Intn(totalRecipeWeight)
		var recipe factory.CatalogComposition
		accum := 0
		for _, comp := range recipes {
			w := comp.Frequency
			if w <= 0 {
				w = 10
			}
			accum += w
			if roll < accum {
				recipe = comp
				break
			}
		}

		// 2. Just pass the requirements straight to the Builder!
		req := factory.GenerationRequest{
			ArchetypeID:            fmt.Sprintf("%s_%03d", recipe.ID, i),
			PersonaRequirements:    recipe.PersonaRequirements,
			RequiredDeviceTags:     recipe.RequiredDeviceTags,
			RequiredWaterSystemTag: recipe.RequiredWaterSystemTag,
			SystemIDs:              recipe.SystemIDs,
			RoutineIDs:             recipe.RoutineIDs,
			AlarmIDs:               recipe.AlarmIDs,
			EventIDs:               recipe.EventIDs,
		}

		node, err := builder.Generate(req)
		if err != nil {
			slog.Warn("Skipping house", slog.Int("index", i), slog.String("recipe", recipe.ID), slog.Any("error", err))
			continue
		}

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
