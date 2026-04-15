package geom

// Transform defines a pure mathematical shift applied to a base value (like a Mean or Max).
// It has zero knowledge of simulation state, weather, or strings.
type Transform struct {
	FlatShift        float64
	ProportionalRate float64 // Multiplier applied to an input delta
	HasMinClamp      bool
	ClampMin         float64
	HasMaxClamp      bool
	ClampMax         float64
}

// Apply takes a base value and an input delta (provided by the engine) and returns the shifted value.
func (t *Transform) Apply(base float64, inputDelta float64) float64 {
	shift := t.FlatShift

	if t.ProportionalRate != 0 {
		proportionalShift := inputDelta * t.ProportionalRate

		if t.HasMinClamp && proportionalShift < t.ClampMin {
			proportionalShift = t.ClampMin
		}
		if t.HasMaxClamp && proportionalShift > t.ClampMax {
			proportionalShift = t.ClampMax
		}

		shift += proportionalShift
	}

	return base + shift
}
