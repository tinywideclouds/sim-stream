package macro_test

import (
	"testing"
	"time"

	"github.com/tinywideclouds/go-power-simulator/internal/simulate/agents/macro"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

func TestLocalizedCalendar_GetDayType(t *testing.T) {
	events := []domain.CalendarEvent{
		{Date: "2026-01-01", Type: "holiday", Description: "Capodanno"},
		{Date: "2026-06-02", Type: "holiday", Description: "Festa della Repubblica"},
	}

	calendar := macro.NewLocalizedCalendar(events)

	tests := []struct {
		name        string
		targetDate  time.Time
		wantDayType string
	}{
		{"Standard Workday", time.Date(2026, time.April, 14, 12, 0, 0, 0, time.UTC), "workday"},
		{"Standard Weekend", time.Date(2026, time.May, 9, 12, 0, 0, 0, time.UTC), "weekend"},
		{"National Holiday", time.Date(2026, time.June, 2, 12, 0, 0, 0, time.UTC), "holiday"},
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
