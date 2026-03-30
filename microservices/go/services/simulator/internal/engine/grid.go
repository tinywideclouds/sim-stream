package engine

import (
	"math"
	"time"

	"github.com/tinywideclouds/go-sim-probability/pkg/generator"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

// GridProvider defines the electrical characteristics of a regional power grid.
type GridProvider interface {
	NominalVoltage() float64
	LiveVoltage(t time.Time) float64
}

// ConfigurableGrid reads its physics directly from the YAML archetype.
type ConfigurableGrid struct {
	template *domain.GridTemplate
	sampler  *generator.Sampler
}

// NewConfigurableGrid creates a live grid based on the YAML definition.
// If no grid is provided in YAML, it defaults to a perfect, flat 230V grid.
func NewConfigurableGrid(template *domain.GridTemplate, sampler *generator.Sampler) *ConfigurableGrid {
	if template == nil {
		// Fallback to a mathematically perfect 230V grid if omitted from YAML
		return &ConfigurableGrid{
			template: &domain.GridTemplate{
				NominalVoltage: 230.0,
				WaveCenter:     230.0,
				WaveAmplitude:  0.0,
				JitterMin:      0.0,
				JitterMax:      0.0,
			},
			sampler: sampler,
		}
	}
	return &ConfigurableGrid{
		template: template,
		sampler:  sampler,
	}
}

func (g *ConfigurableGrid) NominalVoltage() float64 {
	return g.template.NominalVoltage
}

func (g *ConfigurableGrid) LiveVoltage(t time.Time) float64 {
	hourFloat := float64(t.Hour()) + float64(t.Minute())/60.0 + float64(t.Second())/3600.0

	// Apply the Cosine Wave
	angle := (hourFloat - g.template.PeakHour) / 24.0 * 2.0 * math.Pi
	macroSag := g.template.WaveCenter + g.template.WaveAmplitude*math.Cos(angle)

	// Apply the high-frequency Jitter
	jitter, _ := g.sampler.Float64(domain.ProbabilityDistribution{
		Type: domain.DistributionTypeUniform,
		Min:  g.template.JitterMin,
		Max:  g.template.JitterMax,
	})

	return macroSag + jitter
}
