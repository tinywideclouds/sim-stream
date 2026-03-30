package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/tinywideclouds/go-power-simulator/internal/engine"
	mock_engine "github.com/tinywideclouds/go-power-simulator/mock/engine"
	"github.com/tinywideclouds/go-sim-probability/pkg/generator"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

func main() {
	fmt.Println("Starting V2 Multi-Agent Simulation with Pluggable Grid Physics...")

	// 1. Read the V2 Blueprint
	yamlFile, err := os.ReadFile("setup/demo-v2.yaml")
	if err != nil {
		log.Fatalf("Failed to read YAML: %v", err)
	}

	var blueprint domain.NodeArchetype
	err = yaml.Unmarshal(yamlFile, &blueprint)
	if err != nil {
		log.Fatalf("Failed to unmarshal V2 YAML: %v", err)
	}

	// 2. Initialize the V2 Brain & RNG
	var seed [32]byte
	seed[0] = 42

	sampler := generator.NewSampler(seed)
	scheduler := engine.NewScheduler(sampler)
	negotiator := engine.NewNegotiator()
	executor := engine.NewExecutor(sampler)

	// NEW: Create the grid dynamically from the parsed YAML!
	grid := engine.NewConfigurableGrid(blueprint.Grid, sampler)

	// Set Rollover to 4:00 AM
	orchestrator := engine.NewOrchestrator(scheduler, negotiator, executor, 4)

	// 3. Initialize the House Memory (Starts strictly at Midnight)
	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	state := engine.NewSimulationState(&blueprint, startTime)

	// Inject our MockWeather (5.0C day, 0.0C night)
	weather := &mock_engine.MockWeather{StaticTempC: 5.0}

	// 4. Setup CSV Output
	csvFile, err := os.Create("v2_results.csv")
	if err != nil {
		log.Fatalf("Failed to create CSV: %v", err)
	}
	defer csvFile.Close()

	writer := csv.NewWriter(csvFile)
	defer writer.Flush()

	// Write CSV Headers including the new GridVoltage
	writer.Write([]string{
		"Timestamp",
		"GridVoltage",
		"TotalWatts",
		"HotWaterLiters",
		"TankTempC",
		"IndoorTempC",
		"ActiveDevices",
		"ActiveActors",
	})

	// 5. Run the Master Loop (24 hours at 15-second intervals)
	tickDuration := 15 * time.Second
	totalTicks := (24 * time.Hour) / tickDuration

	fmt.Printf("Simulating %d ticks...\n", totalTicks)

	for i := 0; i < int(totalTicks); i++ {
		// Pass the dynamically configured Grid directly into the tick!
		result := orchestrator.Tick(state, tickDuration, weather, grid)

		// UNFILTERED LOGGING: Write every single 15-second tick to the CSV
		row := []string{
			result.Timestamp.Format(time.RFC3339),
			fmt.Sprintf("%.2f", result.GridVoltage),
			fmt.Sprintf("%.2f", result.TotalWatts),
			fmt.Sprintf("%.2f", result.TotalHotLiters),
			fmt.Sprintf("%.2f", result.TankTempC),
			fmt.Sprintf("%.2f", result.IndoorTempC),
			strings.Join(result.ActiveDevices, " | "),
			strings.Join(result.ActiveActors, " | "),
		}
		writer.Write(row)
	}

	fmt.Println("Done! Results written to v2_results.csv")
}
