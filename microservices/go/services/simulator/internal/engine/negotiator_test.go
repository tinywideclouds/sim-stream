package engine_test

import (
	"testing"
	"time"

	"github.com/tinywideclouds/go-power-simulator/internal/engine"
)

func TestNegotiator_ResolveEventTime(t *testing.T) {
	negotiator := engine.NewNegotiator()

	// Base time: 08:00 AM
	baseTime := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		lead       engine.Participant
		dependents []engine.Participant
		expected   time.Time
	}{
		{
			name: "Perfect Alignment (No Friction Needed)",
			lead: engine.Participant{ActorID: "parent_1", TargetTime: baseTime, Weight: 1.0},
			dependents: []engine.Participant{
				{ActorID: "child_1", TargetTime: baseTime, Weight: 0.5},
			},
			expected: baseTime, // 08:00
		},
		{
			name: "The Compromise (Child is late, Parent compromises)",
			// Parent wants 08:00 (Weight 1.0). Child wants 08:30 (Weight 0.5).
			// Math: (0*1.0 + 30*0.5) / 1.5 = +10 minutes.
			lead: engine.Participant{ActorID: "parent_1", TargetTime: baseTime, Weight: 1.0},
			dependents: []engine.Participant{
				{ActorID: "child_1", TargetTime: baseTime.Add(30 * time.Minute), Weight: 0.5},
			},
			expected: baseTime.Add(10 * time.Minute), // 08:10
		},
		{
			name: "The Meltdown (Parent patience snaps)",
			// Parent wants 08:00. Child wants 08:45 (Weight 1.0).
			// Weighted average is 08:22:30.
			// But Parent's patience limit is 10m. They snap at 08:10!
			lead: engine.Participant{ActorID: "parent_1", TargetTime: baseTime, Weight: 1.0, PatienceLimit: 10 * time.Minute},
			dependents: []engine.Participant{
				{ActorID: "child_1", TargetTime: baseTime.Add(45 * time.Minute), Weight: 1.0},
			},
			expected: baseTime.Add(10 * time.Minute), // 08:10 (Forced Abort)
		},
		{
			name: "Multiple Dependents (Two kids delaying the parent)",
			// Parent wants 08:00.
			// Kid 1 wants 08:10 (Weight 0.5).
			// Kid 2 wants 08:20 (Weight 0.5).
			// Math: (0*1.0 + 10*0.5 + 20*0.5) / 2.0 = (5 + 10) / 2 = 7.5 mins.
			lead: engine.Participant{ActorID: "parent_1", TargetTime: baseTime, Weight: 1.0},
			dependents: []engine.Participant{
				{ActorID: "child_1", TargetTime: baseTime.Add(10 * time.Minute), Weight: 0.5},
				{ActorID: "child_2", TargetTime: baseTime.Add(20 * time.Minute), Weight: 0.5},
			},
			expected: baseTime.Add(7*time.Minute + 30*time.Second), // 08:07:30
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := negotiator.ResolveEventTime(tt.lead, tt.dependents)
			if !got.Equal(tt.expected) {
				t.Errorf("ResolveEventTime() = %v, want %v", got, tt.expected)
			}
		})
	}
}
