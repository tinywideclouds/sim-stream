package macro

import (
	"time"

	"github.com/tinywideclouds/go-sim-schema/domain"
)

type CalendarProvider interface {
	GetDayType(date time.Time) string
}

type LocalizedCalendar struct {
	events map[string]string
}

func NewLocalizedCalendar(events []domain.CalendarEvent) *LocalizedCalendar {
	eventMap := make(map[string]string)
	for _, e := range events {
		eventMap[e.Date] = e.Type
	}
	return &LocalizedCalendar{events: eventMap}
}

func (c *LocalizedCalendar) GetDayType(date time.Time) string {
	dateStr := date.Format("2006-01-02")
	if dayType, exists := c.events[dateStr]; exists {
		return dayType
	}
	weekday := date.Weekday()
	if weekday == time.Saturday || weekday == time.Sunday {
		return "weekend"
	}
	return "workday"
}
