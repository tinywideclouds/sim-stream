// lib/profilecompiler/devices.go
package profile

import (
	"github.com/tinywideclouds/go-sim-schema/domain"
)

// GenerateDevices builds the physical hardware profiles for the house.
func GenerateDevices(intent PersonaIntent) []domain.DeviceTemplate {
	var devices []domain.DeviceTemplate

	for _, app := range intent.Appliances {
		switch app {
		case "shower_1":
			devices = append(devices, domain.DeviceTemplate{
				DeviceID: "shower_1",
				// Assuming an electric power shower for high load
				ElectricalProfile: domain.DeviceProfile{
					Type:     domain.ProfileTypeConstant,
					MaxWatts: 8500.0,
				},
				WaterProfile: &domain.WaterProfile{
					ColdLitersPerMinute: 4.0,
					HotLitersPerMinute:  6.0,
				},
			})
		case "cooker_1":
			devices = append(devices, domain.DeviceTemplate{
				DeviceID: "cooker_1",
				ElectricalProfile: domain.DeviceProfile{
					Type:     domain.ProfileTypeConstant,
					MaxWatts: 3000.0, // High draw while cooking
				},
				// Cookers don't draw water, so WaterProfile is naturally nil
			})
		}
	}

	return devices
}
