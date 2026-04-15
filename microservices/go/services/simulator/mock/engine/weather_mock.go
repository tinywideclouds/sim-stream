package mock_engine

import (
	"math"
	"time"
)

// MockWeather provides a dynamic mathematical model of Milan's weather.
type MockWeather struct {
	BaseTempC float64 // Used to globally shift the climate if needed
}

func (m *MockWeather) GetTemperature(t time.Time) float64 {
	// 1. Seasonal Approximation (Coldest around Jan 15, Hottest around mid-July)
	dayOfYear := float64(t.YearDay())
	seasonalOffset := -math.Cos((dayOfYear - 15) / 365.0 * 2.0 * math.Pi)

	// Milan swings from ~4C average in winter to ~25C average in summer
	seasonalTemp := 14.5 + (seasonalOffset * 10.5) + m.BaseTempC

	// 2. Daily Fluctuation (Coldest at 4 AM, Hottest at 4 PM / 16:00)
	hourFloat := float64(t.Hour()) + float64(t.Minute())/60.0
	dailyOffset := -math.Cos((hourFloat - 4.0) / 24.0 * 2.0 * math.Pi)

	// Daily swing is roughly +/- 5 degrees from the daily average
	dailySwing := 5.0

	return seasonalTemp + (dailyOffset * dailySwing)
}

func (m *MockWeather) GetPrecipitation(t time.Time) float64 {
	// Synthetic rain: Predictable based on the day of the month so tests remain deterministic
	month := t.Month()

	// It rains occasionally in Spring and Autumn in Milan
	if month >= time.March && month <= time.May || month >= time.September && month <= time.November {
		if t.Day()%5 == 0 { // Rains every 5 days
			return 2.5 // Light rain (mm)
		}
	}
	return 0.0
}

func (m *MockWeather) GetSolarIrradiance(t time.Time) float64 {
	// Basic solar curve: peaks at noon, zero at night
	hourFloat := float64(t.Hour()) + float64(t.Minute())/60.0

	if hourFloat > 6.0 && hourFloat < 18.0 {
		// Max lux ~80k at noon in summer
		peakLux := 80000.0
		offset := -math.Cos((hourFloat - 6.0) / 12.0 * 2.0 * math.Pi)
		if offset > 0 {
			return peakLux * offset
		}
	}
	return 0.0
}
