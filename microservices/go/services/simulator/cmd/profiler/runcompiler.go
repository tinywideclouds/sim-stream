// cmd/profiler/main.go
package main

import (
	"flag"
	"log"
	"os"

	"github.com/tinywideclouds/go-sim-compiler/profile"
	"gopkg.in/yaml.v3"
)

func main() {
	intentFile := flag.String("intent", "./profiles/basic-intent.yaml", "Input human intent YAML")
	outFile := flag.String("out", "setup/v3-intent-test.yaml", "Output engine-ready YAML")
	strategy := flag.String("ic", "burn-in", "Initial condition strategy (burn-in or algebraic)")
	flag.Parse()

	// 1. Read the simple human intent
	data, err := os.ReadFile(*intentFile)
	if err != nil {
		log.Fatalf("Failed to read intent: %v", err)
	}

	var intent profile.PersonaIntent
	if err := yaml.Unmarshal(data, &intent); err != nil {
		log.Fatalf("Failed to parse intent: %v", err)
	}

	// 2. Run our compiler math (Biology, Pressure, Tuning, ICs)
	archetype := profile.Compile(intent, *strategy)

	// 3. Write the mathematically balanced V3 Engine schema
	outData, err := yaml.Marshal(archetype)
	if err != nil {
		log.Fatalf("Failed to marshal output: %v", err)
	}

	if err := os.WriteFile(*outFile, outData, 0644); err != nil {
		log.Fatalf("Failed to write output: %v", err)
	}

	log.Printf("Topography compiled successfully to %s using %s strategy.", *outFile, *strategy)
}
