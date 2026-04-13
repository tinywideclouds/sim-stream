package engine

// ActorTickState represents the structured output of an AI brain for a single simulation tick.
type ActorTickState struct {
	ActorID  string
	ActionID string
	IsShared bool
	Meters   map[string]float64
}
