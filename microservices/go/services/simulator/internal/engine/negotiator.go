package engine

import (
	"time"
)

// Participant represents a single actor's desires for a collective event.
type Participant struct {
	ActorID       string
	TargetTime    time.Time
	Weight        float64       // Lead is 1.0, Dependents use FrictionWeight (0.0 - 1.0)
	PatienceLimit time.Duration // 0 means infinite patience
}

// Negotiator resolves conflicts between multiple actors for a single event.
type Negotiator struct{}

// NewNegotiator creates a new conflict resolution calculator.
func NewNegotiator() *Negotiator {
	return &Negotiator{}
}

// ResolveEventTime calculates the exact moment a collective event will fire.
func (n *Negotiator) ResolveEventTime(lead Participant, dependents []Participant) time.Time {
	// 1. Calculate the Compromise (Weighted Average of Unix Timestamps)
	totalUnix := float64(lead.TargetTime.Unix()) * lead.Weight
	totalWeight := lead.Weight

	for _, dep := range dependents {
		totalUnix += float64(dep.TargetTime.Unix()) * dep.Weight
		totalWeight += dep.Weight
	}

	negotiatedUnix := totalUnix / totalWeight
	negotiatedTime := time.Unix(int64(negotiatedUnix), 0)

	// 2. Check for Breaking Points (Patience Limits)
	// If the negotiated time makes anyone wait longer than their patience allows,
	// the event is forcefully capped at their snapping point.
	finalTime := negotiatedTime

	// Check the Lead Actor's patience
	if lead.PatienceLimit > 0 {
		snapTime := lead.TargetTime.Add(lead.PatienceLimit)
		if finalTime.After(snapTime) {
			finalTime = snapTime
		}
	}

	// Check all Dependent Actors' patience
	for _, dep := range dependents {
		if dep.PatienceLimit > 0 {
			snapTime := dep.TargetTime.Add(dep.PatienceLimit)
			if finalTime.After(snapTime) {
				finalTime = snapTime
			}
		}
	}

	return finalTime
}
