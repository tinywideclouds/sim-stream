package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/tinywideclouds/go-power-simulator/internal/engine"
	mock_engine "github.com/tinywideclouds/go-power-simulator/mock/engine"
	"github.com/tinywideclouds/go-sim-probability/pkg/generator"
	"github.com/tinywideclouds/go-sim-schema/domain/v1"
)

func main() {
	// 1. Load the Blueprint (YAML)
	blueprintPath := "setup/test-house.yaml"
	blueprint, err := loadBlueprint(blueprintPath)
	if err != nil {
		log.Fatalf("Failed to load blueprint: %v", err)
	}

	// 2. Initialize the Engine Dependencies
	// Use a fixed seed for reproducible test data
	sampler := generator.NewSampler([32]byte{1, 2, 3, 4, 5})

	tickDur := 15 * time.Second
	evaluator := engine.NewEvaluator(sampler, tickDur)
	weather := &mock_engine.MockWeather{StaticTempC: 12.0}

	// 3. Set the Simulation Clock
	// Let's run it from midnight for exactly 1 day
	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := startTime.Add(24 * time.Hour)

	// Initialize the memory state (starts at 15 degrees indoors)
	state := engine.NewSimulationState(blueprint, startTime, 15.0)

	// 4. Prepare the Output CSV
	outFile, err := os.Create("simulation_results.csv")
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer outFile.Close()

	writer := csv.NewWriter(outFile)
	defer writer.Flush()

	// Write CSV Header
	writer.Write([]string{"Timestamp", "IndoorTempC", "TotalWatts", "ActiveEvents"})

	// 5. THE HOT LOOP
	fmt.Printf("Starting simulation for Node: %s\n", blueprint.ArchetypeID)

	ticks := 0
	for state.SimTime.Before(endTime) {
		result, err := engine.Step(state, evaluator, weather, tickDur)
		if err != nil {
			log.Fatalf("Simulation failed at %s: %v", state.SimTime, err)
		}

		// Format output for the CSV
		eventsStr := fmt.Sprintf("%v", result.ActiveEventIDs)
		writer.Write([]string{
			result.SimTime.Format(time.RFC3339),
			fmt.Sprintf("%.2f", result.IndoorTempC),
			fmt.Sprintf("%.2f", result.TotalWatts),
			eventsStr,
		})

		ticks++
	}

	fmt.Printf("Simulation complete! Generated %d rows of 15s tick data.\n", ticks)
	fmt.Printf("Results saved to simulation_results.csv\n")
}

func loadBlueprint(path string) (*domain.NodeArchetype, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var blueprint domain.NodeArchetype
	if err := yaml.Unmarshal(data, &blueprint); err != nil {
		return nil, err
	}

	return &blueprint, nil
}
