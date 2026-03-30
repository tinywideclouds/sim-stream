package generator

import (
	"fmt"
	"math/rand/v2"
	"strconv"
	"time"

	"github.com/tinywideclouds/go-sim-schema/domain"
)

// Sampler wraps the rand/v2 engine to provide reproducible,
// stateful random generation for our simulation ticks.
type Sampler struct {
	rng *rand.Rand
}

// NewSampler creates a new sampler with a ChaCha8 engine.
// Using a fixed seed guarantees reproducible simulation runs.
func NewSampler(seed [32]byte) *Sampler {
	return &Sampler{
		rng: rand.New(rand.NewChaCha8(seed)),
	}
}

// Float64 resolves a probability distribution into a concrete float64.
// It parses the string-based Mean/StdDev into floats on the fly.
func (s *Sampler) Float64(dist domain.ProbabilityDistribution) (float64, error) {
	switch dist.Type {
	case domain.DistributionTypeConstant:
		return strconv.ParseFloat(dist.Value, 64)

	case domain.DistributionTypeUniform:
		// uniform = min + (rand * (max - min))
		val := dist.Min + (s.rng.Float64() * (dist.Max - dist.Min))
		return val, nil

	case domain.DistributionTypeNormal:
		mean, err := strconv.ParseFloat(dist.Mean, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid mean %q for float normal distribution: %w", dist.Mean, err)
		}
		stdDev, err := strconv.ParseFloat(dist.StdDev, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid std_dev %q for float normal distribution: %w", dist.StdDev, err)
		}
		// normal = mean + (randNorm * stdDev)
		val := mean + (s.rng.NormFloat64() * stdDev)
		return val, nil

	default:
		return 0, fmt.Errorf("unsupported distribution type: %v", dist.Type)
	}
}

// Duration resolves a probability distribution into a concrete time.Duration.
// It expects Mean/StdDev/Value to be standard Go duration strings (e.g., "15m", "1h30m").
func (s *Sampler) Duration(dist domain.ProbabilityDistribution) (time.Duration, error) {
	switch dist.Type {
	case domain.DistributionTypeConstant:
		return time.ParseDuration(dist.Value)

	case domain.DistributionTypeUniform:
		// Convert min/max (which are float64 minutes in our domain) to duration
		minD := time.Duration(dist.Min * float64(time.Minute))
		maxD := time.Duration(dist.Max * float64(time.Minute))

		diff := float64(maxD - minD)
		val := float64(minD) + (s.rng.Float64() * diff)
		return time.Duration(val), nil

	case domain.DistributionTypeNormal:
		meanD, err := time.ParseDuration(dist.Mean)
		if err != nil {
			return 0, fmt.Errorf("invalid mean duration %q: %w", dist.Mean, err)
		}
		stdDevD, err := time.ParseDuration(dist.StdDev)
		if err != nil {
			return 0, fmt.Errorf("invalid std_dev duration %q: %w", dist.StdDev, err)
		}

		val := float64(meanD) + (s.rng.NormFloat64() * float64(stdDevD))

		// Durations can't be negative in our physical world (can't boil a kettle for -5 seconds)
		if val < 0 {
			val = 0
		}
		return time.Duration(val), nil

	default:
		return 0, fmt.Errorf("unsupported distribution type for duration: %v", dist.Type)
	}
}
