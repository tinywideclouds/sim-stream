package domain

type WaterProfile struct {
	ColdLitersPerMinute float64 `yaml:"cold_lpm"`
	HotLitersPerMinute  float64 `yaml:"hot_lpm"`
}
