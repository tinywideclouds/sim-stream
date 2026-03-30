package engine

import "time"

// MockWeather provides static weather for our local test runs.
type MockWeather struct {
	StaticTempC float64
}

func (m *MockWeather) GetTemperature(t time.Time) float64 {
	// Let's make it drop 5 degrees at night just to see the heating react
	if t.Hour() < 6 || t.Hour() > 20 {
		return m.StaticTempC - 5.0
	}
	return m.StaticTempC
}

func (m *MockWeather) GetPrecipitation(t time.Time) float64   { return 0.0 }
func (m *MockWeather) GetSolarIrradiance(t time.Time) float64 { return 0.0 }
