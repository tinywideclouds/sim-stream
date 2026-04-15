package probability

import (
	"math"
	"math/rand/v2"
	"time"
)

type Sampler struct {
	randomGenerator *rand.Rand
}

func NewSampler(seed [32]byte) *Sampler {
	return &Sampler{
		randomGenerator: rand.New(rand.NewChaCha8(seed)),
	}
}

type DistributionSampler struct {
	s *Sampler
}

func NewDistributionSampler(s *Sampler) *DistributionSampler {
	return &DistributionSampler{s: s}
}

// Sample executes the math directly. No string parsing required!
func (ds *DistributionSampler) Sample(d SampleSpace) float64 {
	switch d.Type {
	case NormalDistribution:
		return ds.NormalSample(d.Mean, d.StandardDeviation) // FIXED: Was swapped in your sketch
	case UniformDistribution:
		return ds.UniformSample(d.Min, d.Max) // FIXED: Was swapped in your sketch
	case ConstantDistribution:
		return d.Const
	}
	return d.Const
}

// SampleDuration executes the math and cleanly casts back to a Go duration.
func (ds *DistributionSampler) SampleDuration(d SampleSpace) time.Duration {
	val := ds.Sample(d)

	// If the user explicitly provided a multiplier, apply it
	if d.Units.Duration != nil {
		return time.Duration(val) * (*d.Units.Duration)
	}

	// Otherwise, it was already parsed as nanoseconds by UnmarshalYAML
	return time.Duration(val)
}

func (ds *DistributionSampler) UniformSample(a, b float64) float64 {
	max := math.Max(a, b)
	min := math.Min(a, b)
	width := max - min
	return min + ds.s.randomGenerator.Float64()*width
}

func (ds *DistributionSampler) NormalSample(mean, stdDev float64) float64 {
	return mean + ds.s.randomGenerator.NormFloat64()*stdDev
}
