// cmd/runsim/main.go
package main

import (
	crand "crypto/rand"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/tinywideclouds/go-power-simulator/internal/aiengine"
	"github.com/tinywideclouds/go-power-simulator/internal/engine"
	"github.com/tinywideclouds/go-power-simulator/internal/reporting"
	"github.com/tinywideclouds/go-power-simulator/internal/simulate"
	"github.com/tinywideclouds/go-power-simulator/internal/world"
	mock_engine "github.com/tinywideclouds/go-power-simulator/mock/engine"
	"github.com/tinywideclouds/go-sim-probability/pkg/generator"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

func main() {
	inDir := flag.String("dir", "setup/bulk", "Directory containing household YAMLs")
	outDir := flag.String("outdir", "results/test_batch", "Directory to output logs")
	simDays := flag.Float64("days", 14.0, "Days to simulate after burn-in")
	burnInHours := flag.Float64("burn-in", 24.0, "Hours to simulate invisibly to settle initial conditions")
	sampleSize := flag.Int("sample", 100, "Number of households to randomly sample")
	flag.Parse()

	// 1. Ensure Output Directory Exists
	if err := os.MkdirAll(*outDir, 0755); err != nil {
		panic(fmt.Sprintf("Failed to create output directory: %v", err))
	}

	// 2. Setup Centralized Observability (Terminal + File)
	logFile, err := os.Create(filepath.Join(*outDir, "app.log"))
	if err != nil {
		panic(fmt.Sprintf("Failed to create log file: %v", err))
	}
	defer logFile.Close()

	logger := slog.New(slog.NewTextHandler(io.MultiWriter(os.Stdout, logFile), &slog.HandlerOptions{Level: slog.LevelWarn}))
	slog.SetDefault(logger)

	// 3. Initialize the Global CSV Reporter
	reporter, err := reporting.NewCSVReporter(*outDir)
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize CSV reporter: %v", err))
	}
	defer reporter.Close()

	// 4. Load Payload Files
	files, err := os.ReadDir(*inDir)
	if err != nil {
		slog.Error("Failed to read input directory", "dir", *inDir, "error", err)
		os.Exit(1)
	}

	var validFiles []string
	for _, f := range files {
		if !f.IsDir() && (filepath.Ext(f.Name()) == ".yaml" || filepath.Ext(f.Name()) == ".yml") {
			validFiles = append(validFiles, filepath.Join(*inDir, f.Name()))
		}
	}

	if len(validFiles) == 0 {
		slog.Error("No YAML files found in input directory", "dir", *inDir)
		os.Exit(1)
	}

	// Shuffle and Sample
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	rng.Shuffle(len(validFiles), func(i, j int) {
		validFiles[i], validFiles[j] = validFiles[j], validFiles[i]
	})
	if *sampleSize < len(validFiles) {
		validFiles = validFiles[:*sampleSize]
	}

	// 5. Shared Providers & Math Constraints
	weatherProvider := &mock_engine.MockWeather{StaticTempC: 5.0}

	samplingInterval := 15 * time.Second

	burnInDuration := time.Duration(*burnInHours*3600) * time.Second
	simulationDuration := time.Duration(*simDays*24*3600) * time.Second

	// 6. Batch Execution Loop
	for i, file := range validFiles {
		data, err := os.ReadFile(file)
		if err != nil {
			slog.Error("Failed to read file", "file", file, "error", err)
			continue
		}

		var blueprint domain.NodeArchetype
		if err := yaml.Unmarshal(data, &blueprint); err != nil {
			slog.Error("Failed to parse YAML", "file", file, "error", err)
			continue
		}

		slog.Info(fmt.Sprintf("=== Simulating [%d/%d]: %s ===", i+1, len(validFiles), blueprint.ArchetypeID))

		var seed [32]byte
		crand.Read(seed[:])
		sampler := generator.NewSampler(seed)

		calendar := world.NewLocalizedCalendar([]domain.CalendarEvent{})
		gridProvider := engine.NewConfigurableGrid(blueprint.Grid, sampler)

		// Instantiate AI Engines
		utilityBrain := aiengine.NewUtilityEngine(sampler)
		routineBrain := aiengine.NewRoutineEngine(sampler, 3) // 3 AM rollover

		stableEngine := world.NewStableEngine(utilityBrain, routineBrain, calendar, sampler)
		orchestrator := aiengine.NewOrchestrator(stableEngine, sampler)

		startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		state := engine.NewSimulationState(&blueprint, startTime)

		// --- PHASE 1: BURN-IN ---
		// We pass 'nil' for all four reporters so this period generates no UI output.
		slog.Info("Starting burn-in phase...", "household", blueprint.ArchetypeID, "duration", burnInDuration)
		burnInRunner := simulate.NewRunner(orchestrator, nil, nil, nil)

		if err := burnInRunner.Run(state, burnInDuration, samplingInterval, samplingInterval, weatherProvider, gridProvider); err != nil {
			slog.Error("Burn-in failed", "household", blueprint.ArchetypeID, "error", err)
			continue
		}

		// --- PHASE 2: PRIMARY SIMULATION ---
		telemetryInterval := 5 * time.Minute
		slog.Info("Starting main simulation...", "household", blueprint.ArchetypeID, "duration", simulationDuration)
		// We pass the global CSV reporter to all three data stream arguments
		mainRunner := simulate.NewRunner(orchestrator, reporter, reporter, reporter)
		if err := mainRunner.Run(state, simulationDuration, samplingInterval, telemetryInterval, weatherProvider, gridProvider); err != nil {
			slog.Error("Simulation failed", "household", blueprint.ArchetypeID, "error", err)
			continue
		}

		slog.Info("Simulation complete.", "household", blueprint.ArchetypeID)
	}

	slog.Info("Batch processing complete.", "total_households", len(validFiles))
}
