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

	mock_engine "github.com/tinywideclouds/go-power-simulator/mock/engine"
	"github.com/tinywideclouds/go-sim-schema/domain"

	"github.com/tinywideclouds/go-maths/pkg/probability"
	"github.com/tinywideclouds/go-power-simulator/internal/reporting"

	"github.com/tinywideclouds/go-power-simulator/internal/simulate/agents/macro"
	"github.com/tinywideclouds/go-power-simulator/internal/simulate/agents/micro"
	"github.com/tinywideclouds/go-power-simulator/internal/simulate/core"
	"github.com/tinywideclouds/go-power-simulator/internal/simulate/orchestrator"
	"github.com/tinywideclouds/go-power-simulator/internal/simulate/runner"
)

func main() {
	inDir := flag.String("dir", "setup/bulk", "Directory containing household YAMLs")
	outDir := flag.String("outdir", "results/test_batch", "Directory to output logs")
	simDays := flag.Float64("days", 14.0, "Days to simulate after burn-in")
	burnInHours := flag.Float64("burn-in", 24.0, "Hours to simulate invisibly to settle initial conditions")
	sampleSize := flag.Int("sample", 100, "Number of households to randomly sample")
	flag.Parse()

	if err := os.MkdirAll(*outDir, 0755); err != nil {
		panic(fmt.Sprintf("Failed to create output directory: %v", err))
	}

	logFile, err := os.Create(filepath.Join(*outDir, "app.log"))
	if err != nil {
		panic(fmt.Sprintf("Failed to create log file: %v", err))
	}
	defer logFile.Close()

	logger := slog.New(slog.NewTextHandler(io.MultiWriter(os.Stdout, logFile), &slog.HandlerOptions{Level: slog.LevelWarn}))
	slog.SetDefault(logger)

	reporter, err := reporting.NewCSVReporter(*outDir)
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize CSV reporter: %v", err))
	}
	defer reporter.Close()

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

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	rng.Shuffle(len(validFiles), func(i, j int) {
		validFiles[i], validFiles[j] = validFiles[j], validFiles[i]
	})
	if *sampleSize < len(validFiles) {
		validFiles = validFiles[:*sampleSize]
	}

	weatherProvider := &mock_engine.MockWeather{}
	samplingInterval := 15 * time.Second
	burnInDuration := time.Duration(*burnInHours*3600) * time.Second
	simulationDuration := time.Duration(*simDays*24*3600) * time.Second

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

		// 0. Initialize the new go-maths sampler
		var seed [32]byte
		crand.Read(seed[:])
		baseSampler := probability.NewSampler(seed)
		distSampler := probability.NewDistributionSampler(baseSampler)

		gridProvider := core.NewConfigurableGrid(blueprint.Grid, distSampler)

		// 1. Initialize the Micro Brains First (Utility is needed by StableEngine)
		utilityBrain := micro.NewUtilityEngine(distSampler)
		routineBrain := micro.NewRoutineEngine(distSampler, 3)

		// 2. Initialize the Macro Brains
		calendar := macro.NewLocalizedCalendar([]domain.CalendarEvent{})
		stableEngine := macro.NewStableEngine(utilityBrain, calendar, distSampler)

		// 3. Initialize the Orchestrator (The Conductor)
		orch := orchestrator.NewOrchestrator(stableEngine, utilityBrain, routineBrain, distSampler)

		// 4. Build the initial state
		startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		state := core.NewSimulationState(&blueprint, startTime)

		// --- PHASE 1: BURN-IN ---
		slog.Info("Starting burn-in phase...", "household", blueprint.ArchetypeID, "duration", burnInDuration)
		burnInRunner := runner.NewRunner(orch, nil, nil, nil)
		if err := burnInRunner.Run(state, burnInDuration, samplingInterval, samplingInterval, weatherProvider, gridProvider); err != nil {
			slog.Error("Burn-in failed", "household", blueprint.ArchetypeID, "error", err)
			continue
		}

		// --- PHASE 2: PRIMARY SIMULATION ---
		telemetryInterval := 5 * time.Minute
		slog.Info("Starting main simulation...", "household", blueprint.ArchetypeID, "duration", simulationDuration)
		mainRunner := runner.NewRunner(orch, reporter, reporter, reporter)
		if err := mainRunner.Run(state, simulationDuration, samplingInterval, telemetryInterval, weatherProvider, gridProvider); err != nil {
			slog.Error("Simulation failed", "household", blueprint.ArchetypeID, "error", err)
			continue
		}

		slog.Info("Simulation complete.", "household", blueprint.ArchetypeID)
	}

	slog.Info("Batch processing complete.", "total_households", len(validFiles))
}
