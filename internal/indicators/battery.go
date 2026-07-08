//go:build windows

package indicators

import "github.com/fourhundredfour/pinky/internal/win32"

// SYSTEM_POWER_STATUS.BatteryFlag bits/sentinels (winbase.h).
const (
	batteryFlagHigh        = 0x01
	batteryFlagLow         = 0x02
	batteryFlagCritical    = 0x04
	batteryFlagCharging    = 0x08
	batteryFlagNoBattery   = 0x80
	batteryFlagUnknown     = 0xFF
	batteryLifePercentUnknown = 0xFF
)

// Battery is the payload sent to the frontend for the battery indicator.
type Battery struct {
	Present  bool `json:"present"`
	Percent  int  `json:"percent"`
	Charging bool `json:"charging"`
}

// readBattery reports the current battery state, or Present=false on
// desktops (no battery) or if the query fails.
func readBattery() Battery {
	status, ok := win32.GetSystemPowerStatus()
	if !ok {
		return Battery{}
	}
	flag := status.BatteryFlag
	if flag == batteryFlagNoBattery || flag == batteryFlagUnknown {
		return Battery{}
	}

	percent := int(status.BatteryLifePercent)
	if status.BatteryLifePercent == batteryLifePercentUnknown {
		percent = -1
	}
	return Battery{
		Present:  true,
		Percent:  percent,
		Charging: flag&batteryFlagCharging != 0,
	}
}
