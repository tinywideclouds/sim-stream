// runsim.go
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/tinywideclouds/go-power-simulator/internal/aiengine"
	"github.com/tinywideclouds/go-power-simulator/internal/engine"
	mock_engine "github.com/tinywideclouds/go-power-simulator/mock/engine"
	"github.com/tinywideclouds/go-sim-probability/pkg/generator"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

func main() {
	// NEW: Add CLI flags for flexible compiler testing
	inFile := flag.String("in", "setup/v3-intent-test.yaml", "Path to input YAML")
	outFile := flag.String("out", "v3_intent_results.csv", "Path to output CSV")
	burnInHours := flag.Float64("burn-in", 0.0, "Hours to simulate invisibly to settle initial conditions")
	simDays := flag.Float64("days", 1.0, "Days to record to CSV after burn-in")
	flag.Parse()

	fmt.Printf("Starting Pipeline Simulation...\nInput: %s\nBurn-in: %.1fh | Recording: %.1fd\n", *inFile, *burnInHours, *simDays)

	yamlFile, err := os.ReadFile(*inFile)
	if err != nil {
		log.Fatalf("Failed to read YAML: %v", err)
	}

	var blueprint domain.NodeArchetype
	err = yaml.Unmarshal(yamlFile, &blueprint)
	if err != nil {
		log.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	var seed [32]byte
	rand.Seed(time.Now().UnixNano())
	rand.Read(seed[:])
	sampler := generator.NewSampler(seed)

	utilityBrain := aiengine.NewUtilityEngine(sampler)
	orchestrator := aiengine.NewOrchestrator(utilityBrain, sampler)

	grid := engine.NewConfigurableGrid(blueprint.Grid, sampler)
	weather := &mock_engine.MockWeather{StaticTempC: 5.0}

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	state := engine.NewSimulationState(&blueprint, startTime)

	csvFile, err := os.Create(*outFile)
	if err != nil {
		log.Fatalf("Failed to create CSV: %v", err)
	}
	defer csvFile.Close()

	writer := csv.NewWriter(csvFile)
	defer writer.Flush()

	writer.Write([]string{
		"Timestamp",
		"ActiveDevices",
		"ActiveActors",
		"Anomalies",
		"Debug",
	})

	tickDuration := 15 * time.Second
	burnInDuration := time.Duration(*burnInHours * float64(time.Hour))
	recordDuration := time.Duration(*simDays * 24.0 * float64(time.Hour))
	totalDuration := burnInDuration + recordDuration
	totalTicks := totalDuration / tickDuration

	for i := 0; i < int(totalTicks); i++ {
		result := orchestrator.Tick(state, tickDuration, weather, grid)

		// NEW: Only record to CSV if we have passed the burn-in period
		elapsed := state.SimTime.Sub(startTime)
		if elapsed >= burnInDuration {
			var validAnomalies []string
			for _, a := range result.Anomalies {
				if a != "" {
					validAnomalies = append(validAnomalies, a)
				}
			}

			writer.Write([]string{
				result.Timestamp.Format("15:04:05"),
				strings.Join(result.ActiveDevices, "|"),
				strings.Join(result.ActiveActors, "|"),
				strings.Join(validAnomalies, "|"),
				strings.Join(result.DebugLog, " | "),
			})
		}
	}
	fmt.Println("Simulation Complete.")
}
