// internal/world/calendar_test.go
package world_test

import (
	"testing"
	"time"

	"github.com/tinywideclouds/go-power-simulator/internal/world"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

func TestLocalizedCalendar_GetDayType(t *testing.T) {
	// A mock localized YAML payload for Turin, Italy in 2026
	events := []domain.CalendarEvent{
		{Date: "2026-01-01", DayType: "holiday", Description: "Capodanno"},
		{Date: "2026-06-02", DayType: "holiday", Description: "Festa della Repubblica"},
		{Date: "2026-06-24", DayType: "holiday", Description: "Festa di San Giovanni (Turin)"},
	}

	calendar := world.NewLocalizedCalendar(events)

	tests := []struct {
		name        string
		targetDate  time.Time
		wantDayType string
	}{
		{
			name:        "Standard Workday (Tuesday in April)",
			targetDate:  time.Date(2026, time.April, 14, 12, 0, 0, 0, time.UTC),
			wantDayType: "workday",
		},
		{
			name:        "Standard Weekend (Saturday in May)",
			targetDate:  time.Date(2026, time.May, 9, 12, 0, 0, 0, time.UTC),
			wantDayType: "weekend",
		},
		{
			name:        "National Holiday (Festa della Repubblica)",
			targetDate:  time.Date(2026, time.June, 2, 12, 0, 0, 0, time.UTC),
			wantDayType: "holiday",
		},
		{
			name:        "Local Patron Saint Holiday (San Giovanni)",
			targetDate:  time.Date(2026, time.June, 24, 12, 0, 0, 0, time.UTC),
			wantDayType: "holiday",
		},
		{
			name:        "Holiday falling on a weekend (Capodanno is a Thursday in 2026, but let's test override precedence)",
			targetDate:  time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC),
			wantDayType: "holiday", // Should override any underlying weekday math
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calendar.GetDayType(tt.targetDate)
			if got != tt.wantDayType {
				t.Errorf("GetDayType() = %v, want %v", got, tt.wantDayType)
			}
		})
	}
}
