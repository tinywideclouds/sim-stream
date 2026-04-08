// internal/world/calendar.go
package world

import (
	"time"

	"github.com/tinywideclouds/go-sim-schema/domain"
)

// CalendarProvider defines the interface for societal macro-scheduling.
type CalendarProvider interface {
	GetDayType(date time.Time) string
}

// LocalizedCalendar evaluates real-world time against a set of localized YAML rules.
type LocalizedCalendar struct {
	events map[string]string // Maps YYYY-MM-DD to the day_type (e.g., "holiday")
}

// NewLocalizedCalendar initializes a calendar with the parsed domain events.
func NewLocalizedCalendar(events []domain.CalendarEvent) *LocalizedCalendar {
	eventMap := make(map[string]string)
	for _, e := range events {
		eventMap[e.Date] = e.DayType
	}
	return &LocalizedCalendar{events: eventMap}
}

// GetDayType resolves the chronological day into a societal day type.
// Hierarchy: Exact Date Match -> Weekend Rule -> Default Workday.
func (c *LocalizedCalendar) GetDayType(date time.Time) string {
	dateStr := date.Format("2006-01-02")

	// 1. Check for specific overrides (Holidays, forced days off)
	if dayType, exists := c.events[dateStr]; exists {
		return dayType
	}

	// 2. Check for standard societal weekends
	weekday := date.Weekday()
	if weekday == time.Saturday || weekday == time.Sunday {
		return "weekend"
	}

	// 3. Default to a standard workday
	return "workday"
}
