package generator_test

import (
	"math"
	"testing"
	"time"

	"github.com/tinywideclouds/go-sim-probability/pkg/generator"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

func TestSampler_Float64(t *testing.T) {
	// A fixed seed means these "random" numbers will be the exact same every time the test runs.
	seed := [32]byte{1, 2, 3, 4, 5}
	sampler := generator.NewSampler(seed)

	tests := []struct {
		name    string
		dist    domain.ProbabilityDistribution
		wantMin float64 // For uniform bounds checking
		wantMax float64
	}{
		{
			name: "Constant Value",
			dist: domain.ProbabilityDistribution{
				Type:  domain.DistributionTypeConstant,
				Value: "42.5",
			},
			wantMin: 42.5,
			wantMax: 42.5,
		},
		{
			name: "Uniform Distribution (10 to 20)",
			dist: domain.ProbabilityDistribution{
				Type: domain.DistributionTypeUniform,
				Min:  10.0,
				Max:  20.0,
			},
			wantMin: 10.0,
			wantMax: 20.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run it 100 times to ensure it stays within bounds
			for i := 0; i < 100; i++ {
				got, err := sampler.Float64(tt.dist)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				// Floating point math needs a tiny epsilon for equality checks
				if got < tt.wantMin-0.0001 || got > tt.wantMax+0.0001 {
					t.Errorf("Float64() = %v, wanted between %v and %v", got, tt.wantMin, tt.wantMax)
				}
			}
		})
	}
}

func TestSampler_Duration_NormalDistribution(t *testing.T) {
	seed := [32]byte{9, 9, 9}
	sampler := generator.NewSampler(seed)

	dist := domain.ProbabilityDistribution{
		Type:   domain.DistributionTypeNormal,
		Mean:   "15m",
		StdDev: "2m",
	}

	// We'll calculate the average of 10,000 runs.
	// By the Law of Large Numbers, it should converge very closely to 15m.
	var total time.Duration
	iterations := 10000

	for i := 0; i < iterations; i++ {
		val, err := sampler.Duration(dist)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		total += val
	}

	avg := time.Duration(int64(total) / int64(iterations))
	expected := 15 * time.Minute

	// Allow a 5 second tolerance for the statistical variance
	tolerance := 5 * time.Second
	diff := time.Duration(math.Abs(float64(avg - expected)))

	if diff > tolerance {
		t.Errorf("Normal distribution failed to converge. Expected ~%v, got %v", expected, avg)
	}
}
