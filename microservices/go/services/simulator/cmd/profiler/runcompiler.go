package main

import (
	"flag"
	"fmt"
	"log"
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

	log.Printf("--- Starting Bulk Household Profiler ---")

	// 1. Load the Registry
	reg, err := factory.LoadRegistry(*catalogsDir)
	if err != nil {
		log.Fatalf("Critical: Failed to load catalog registry: %v", err)
	}

	if len(reg.Compositions) == 0 {
		log.Fatalf("Critical: No household compositions found in %s/compositions.", *catalogsDir)
	}

	log.Printf("Registry loaded: %d Devices, %d Actions, %d Personas, %d Recipes",
		len(reg.Devices), len(reg.Actions), len(reg.Personas), len(reg.Compositions))

	// 2. Prepare Output Environment
	if err := os.MkdirAll(*outDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
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
			RequiredWaterSystemTag: recipe.RequiredWaterSystemTag, // Also ensuring water systems map correctly
			SystemIDs:              recipe.SystemIDs,
			RoutineIDs:             recipe.RoutineIDs,
			AlarmIDs:               recipe.AlarmIDs,
			EventIDs:               recipe.EventIDs,
		}

		// RESOLVE DYNAMIC PERSONA REQUIREMENTS
		for _, preq := range recipe.PersonaRequirements {
			available := personasByType[preq.Type]
			if len(available) == 0 {
				// Only warn if the minimum is greater than 0
				if preq.Min > 0 {
					log.Printf("Warning: Recipe %s requires type '%s' but none exist.", recipe.ID, preq.Type)
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
			log.Printf("Warning: Skipping house %03d [%s]: %v", i, recipe.ID, err)
			continue
		}

		// Serialize & Write
		outData, err := yaml.Marshal(node)
		if err != nil {
			log.Fatalf("Failed to marshal YAML for %s: %v", req.ArchetypeID, err)
		}

		fileName := fmt.Sprintf("%s.yaml", req.ArchetypeID)
		filePath := filepath.Join(*outDir, fileName)

		if err := os.WriteFile(filePath, outData, 0644); err != nil {
			log.Fatalf("Failed to write file %s: %v", filePath, err)
		}

		successCount++
	}

	log.Printf("--- Profiler Finished ---")
	log.Printf("Successfully generated %d unique households in %s.", successCount, *outDir)
}
