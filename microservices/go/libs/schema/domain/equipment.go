package domain

import (
	"github.com/tinywideclouds/go-maths/pkg/probability"
)

// DeviceProfile defines the electrical load characteristics of a device.
type DeviceProfile struct {
	Type             ProfileType             `yaml:"type"`
	MaxWatts         float64                 `yaml:"max_watts"`
	StandbyWatts     float64                 `yaml:"standby_watts"`
	CooldownDuration probability.SampleSpace `yaml:"cooldown_duration"`
}

type ThermalProfile struct {
	RadiatedWatts float64 `yaml:"radiated_watts"`
}

// WaterProfile defines the plumbing load characteristics, splitting hot and cold.
type WaterProfile struct {
	ColdLitersPerMinute float64 `yaml:"cold_lpm"`
	HotLitersPerMinute  float64 `yaml:"hot_lpm"`
}

// DeviceTaxonomy categorizes a device for reporting and ML aggregation.
type DeviceTaxonomy struct {
	Category  DeviceCategory `yaml:"category"`
	ClassName string         `yaml:"class_name"`
}

// DeviceTemplate is the physical hardware definition inside the simulation node.
type DeviceTemplate struct {
	DeviceID          string            `yaml:"device_id"`
	Taxonomy          DeviceTaxonomy    `yaml:"taxonomy"`
	Specifics         map[string]string `yaml:"specifics"`
	ElectricalProfile DeviceProfile     `yaml:"electrical_profile"`
	WaterProfile      *WaterProfile     `yaml:"water_profile" `
	ThermalProfile    *ThermalProfile   `yaml:"thermal_profile"`
}
