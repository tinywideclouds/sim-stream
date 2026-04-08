// cmd/runsim/main.go
package main

import (
	crand "crypto/rand"
	"encoding/csv"
	"flag"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/tinywideclouds/go-power-simulator/internal/aiengine"
	"github.com/tinywideclouds/go-power-simulator/internal/engine"
	"github.com/tinywideclouds/go-power-simulator/internal/world"
	mock_engine "github.com/tinywideclouds/go-power-simulator/mock/engine"
	"github.com/tinywideclouds/go-sim-probability/pkg/generator"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

func main() {
	inDir := flag.String("dir", "setup/bulk", "Directory containing household YAMLs")
	outDir := flag.String("outdir", "results/test_batch", "Directory to output logs")
	simDays := flag.Float64("days", 7.0, "Days to simulate after burn-in")
	burnInHours := flag.Float64("burn-in", 24.0, "Hours to simulate invisibly to settle initial conditions")
	sampleSize := flag.Int("sample", 1, "Number of random households to sample (0 = run all)")
	logLevel := flag.String("loglevel", "info", "Log level: debug, info, warn, error")
	flag.Parse()

	// 1. Setup Structured Logger
	var level slog.Level
	switch strings.ToLower(*logLevel) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	slog.Info("Starting Batch Simulation", "days", *simDays, "burn_in_hours", *burnInHours)
	slog.Info("Directories configured", "input", *inDir, "output", *outDir)

	if err := os.MkdirAll(*outDir, 0755); err != nil {
		slog.Error("Failed to create output directory", "error", err)
		os.Exit(1)
	}

	// 2. Setup the Aggregate Log (Shared across all houses)
	aggFile, err := os.Create(filepath.Join(*outDir, "_aggregate_summary.csv"))
	if err != nil {
		slog.Error("Failed to create aggregate log", "error", err)
		os.Exit(1)
	}
	defer aggFile.Close()
	aggWriter := csv.NewWriter(aggFile)
	defer aggWriter.Flush()

	aggWriter.Write([]string{
		"HouseholdID", "SimDay", "Timestamp", "EventType", "OccupantsHome", "ActiveDevices", "AnomaliesToday",
	})

	// 3. Find all YAMLs in the target directory
	files, err := os.ReadDir(*inDir)
	if err != nil {
		slog.Error("Failed to read input directory", "error", err)
		os.Exit(1)
	}

	var yamlFiles []os.DirEntry
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".yaml") {
			yamlFiles = append(yamlFiles, file)
		}
	}

	if len(yamlFiles) == 0 {
		slog.Error("No YAML files found", "directory", *inDir)
		os.Exit(1)
	}

	// 4. Randomly Sample the Files using localized rng
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	if *sampleSize > 0 && *sampleSize < len(yamlFiles) {
		slog.Info("Randomly sampling households", "sample_size", *sampleSize, "total_available", len(yamlFiles))
		rng.Shuffle(len(yamlFiles), func(i, j int) {
			yamlFiles[i], yamlFiles[j] = yamlFiles[j], yamlFiles[i]
		})
		yamlFiles = yamlFiles[:*sampleSize]
	} else {
		slog.Info("Running all households", "count", len(yamlFiles))
	}

	// 5. Run the simulations
	for _, file := range yamlFiles {
		runHouseholdSimulation(filepath.Join(*inDir, file.Name()), *outDir, *simDays, *burnInHours, aggWriter)
	}

	slog.Info("Batch Simulation Complete. Check aggregate summary for macro trends.")
}

func runHouseholdSimulation(yamlPath, outDir string, days, burnInHours float64, aggWriter *csv.Writer) {
	slog.Debug("Simulating household", "yaml_path", yamlPath)

	yamlFile, err := os.ReadFile(yamlPath)
	if err != nil {
		slog.Error("Failed to read YAML", "yaml_path", yamlPath, "error", err)
		return
	}

	var blueprint domain.NodeArchetype
	if err := yaml.Unmarshal(yamlFile, &blueprint); err != nil {
		slog.Error("Failed to unmarshal YAML", "yaml_path", yamlPath, "error", err)
		return
	}

	// Setup Engine Components securely using crypto/rand for the probability seed
	var seed [32]byte
	if _, err := crand.Read(seed[:]); err != nil {
		slog.Error("Failed to generate secure random seed", "error", err)
		return
	}
	sampler := generator.NewSampler(seed)

	// --- THE FIX: WIRED HYBRID ENGINE ---
	// 1. Setup V3 Brain
	utilityBrain := aiengine.NewUtilityEngine(sampler)

	// 2. Setup V2 Brain
	scheduler := engine.NewScheduler(sampler)
	negotiator := engine.NewNegotiator()
	executor := engine.NewExecutor(sampler)
	routineBrain := aiengine.NewRoutineEngine(scheduler, negotiator, executor, 3) // 3 AM rollover

	// 3. Setup Macro Calendar
	cal := world.NewLocalizedCalendar([]domain.CalendarEvent{})

	// 4. Setup the Arbiter
	arbiter := world.NewStableEngine(utilityBrain, routineBrain, cal, sampler)

	// 5. Inject Arbiter into Orchestrator
	orchestrator := aiengine.NewOrchestrator(arbiter, sampler)
	// ------------------------------------

	grid := engine.NewConfigurableGrid(blueprint.Grid, sampler)
	weather := &mock_engine.MockWeather{StaticTempC: 5.0}

	// Start on a Monday (Jan 5, 2026) so we can clearly see weekends
	startTime := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	state := engine.NewSimulationState(&blueprint, startTime)

	// Setup Detailed Log
	detFile, err := os.Create(filepath.Join(outDir, fmt.Sprintf("%s_detailed.csv", blueprint.ArchetypeID)))
	if err != nil {
		slog.Error("Failed to create detailed CSV", "error", err)
		return
	}
	defer detFile.Close()
	detWriter := csv.NewWriter(detFile)
	defer detWriter.Flush()

	detWriter.Write([]string{"Timestamp", "ActiveDevices", "ActiveActors", "Anomalies", "Debug"})

	// Tick math
	tickDuration := 15 * time.Second
	burnInDuration := time.Duration(burnInHours * float64(time.Hour))
	recordDuration := time.Duration(days * 24.0 * float64(time.Hour))
	totalDuration := burnInDuration + recordDuration
	totalTicks := totalDuration / tickDuration

	prevActorsState := ""
	anomaliesToday := 0
	currentDay := startTime.Day()
	hasLoggedStart := false

	for i := 0; i < int(totalTicks); i++ {
		result := orchestrator.Tick(state, tickDuration, weather, grid)

		// Burn-in check
		elapsed := state.SimTime.Sub(startTime)
		if elapsed < burnInDuration {
			// Silently update day tracker to prevent immediate "End_Of_Day" log on first recorded tick
			currentDay = result.Timestamp.Day()
			continue
		}

		// --- Everything below this line only runs AFTER burn-in ---

		// 1. Process Detailed Log
		var validAnomalies []string
		for _, a := range result.Anomalies {
			if a != "" {
				validAnomalies = append(validAnomalies, a)
				anomaliesToday++
			}
		}

		currentActorsState := strings.Join(result.ActiveActors, "|")
		currentDevices := strings.Join(result.ActiveDevices, "|")
		timeStr := result.Timestamp.Format("2006-01-02 15:04:05")

		detWriter.Write([]string{
			timeStr, currentDevices, currentActorsState, strings.Join(validAnomalies, "|"), strings.Join(result.DebugLog, " | "),
		})

		// 2. Process Aggregate Log: Transition Check
		if currentActorsState != prevActorsState || !hasLoggedStart {
			eventType := "Transition"
			if !hasLoggedStart {
				eventType = "Sim_Start"
				hasLoggedStart = true
			}
			aggWriter.Write([]string{
				blueprint.ArchetypeID,
				fmt.Sprintf("%d", result.Timestamp.YearDay()),
				timeStr,
				eventType,
				currentActorsState,
				currentDevices,
				"-",
			})
			prevActorsState = currentActorsState
		}

		// 3. Process Aggregate Log: End of Day Check
		if result.Timestamp.Day() != currentDay {
			aggWriter.Write([]string{
				blueprint.ArchetypeID,
				fmt.Sprintf("%d", currentDay),
				result.Timestamp.Format("2006-01-02 00:00:00"),
				"End_Of_Day_Summary",
				currentActorsState,
				fmt.Sprintf("Total Anomalies: %d", anomaliesToday),
				fmt.Sprintf("%d", anomaliesToday),
			})
			currentDay = result.Timestamp.Day()
			anomaliesToday = 0

			aggWriter.Flush()
			detWriter.Flush()
		}
	}

	slog.Info("Completed household", "yaml_path", yamlPath)
}
