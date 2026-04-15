package geom

import "math"

type CurveType string

const (
	Linear      CurveType = "linear"
	EaseIn      CurveType = "ease_in"  // Slow start, fast finish (Back-loaded)
	EaseOut     CurveType = "ease_out" // Fast start, slow finish (Front-loaded)
	Bell        CurveType = "bell"     // Fast middle, slow edges (S-Curve)
	Exponential CurveType = "exponential"
)

// Evaluate returns the Y value (0.0 to 1.0) for a given X progression (0.0 to 1.0).
// Useful for determining instant urgency or probability weighting.
func EvaluateCurve(curve CurveType, p float64) float64 {
	if p <= 0.0 {
		return 0.0
	}
	if p >= 1.0 {
		return 1.0
	}

	switch curve {
	case EaseOut:
		return 1.0 - math.Pow(1.0-p, 3)
	case EaseIn, Exponential:
		return math.Pow(p, 3)
	case Bell:
		return p * p * (3.0 - 2.0*p)
	case Linear:
		fallthrough
	default:
		return p
	}
}

// Integral returns the accumulated area under the curve (0.0 to 1.0).
// Useful for calculating how much of a biological need is satisfied if an action is interrupted halfway.
func CurveIntegral(curve CurveType, p float64) float64 {
	if p <= 0.0 {
		return 0.0
	}
	if p >= 1.0 {
		return 1.0
	}

	switch curve {
	case EaseIn:
		return p * p // Simplified integral representation
	case EaseOut:
		return 1.0 - math.Pow(1.0-p, 2)
	case Bell:
		return p * p * (3.0 - 2.0*p)
	case Linear:
		fallthrough
	default:
		return p
	}
}
