package core

import (
	"math"
	"time"

	"github.com/tinywideclouds/go-maths/pkg/probability"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

type GridProvider interface {
	NominalVoltage() float64
	LiveVoltage(t time.Time) float64
}

type ConfigurableGrid struct {
	template *domain.GridTemplate
	sampler  *probability.DistributionSampler
}

func NewConfigurableGrid(template *domain.GridTemplate, sampler *probability.DistributionSampler) *ConfigurableGrid {
	if template == nil {
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

	angle := (hourFloat - g.template.PeakHour) / 24.0 * 2.0 * math.Pi
	macroSag := g.template.WaveCenter + g.template.WaveAmplitude*math.Cos(angle)

	// Pure Math Jitter!
	jitterDist := probability.SampleSpace{
		Type: probability.UniformDistribution,
		Min:  g.template.JitterMin,
		Max:  g.template.JitterMax,
	}
	jitter := g.sampler.Sample(jitterDist)

	return macroSag + jitter
}
