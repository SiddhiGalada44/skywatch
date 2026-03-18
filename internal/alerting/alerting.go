package alerting

import (
	"fmt"
	"time"

	"skywatch/internal/telemetry"
)

// CheckAlerts checks for alerts based on new telemetry and previous state.
func CheckAlerts(t telemetry.Telemetry, previousAltitude float64, previousAltitudeTime time.Time) []telemetry.Alert {
	var alerts []telemetry.Alert

	// Low battery alert
	if t.BatteryPct < 20 {
		alerts = append(alerts, telemetry.Alert{
			ID:        telemetry.GenerateAlertID(),
			VehicleID: t.VehicleID,
			Type:      "low_battery",
			Message:   fmt.Sprintf("Battery low: %.1f%%", t.BatteryPct),
			Timestamp: time.Now(),
		})
	}

	// Altitude drop alert
	if previousAltitude > 0 && !previousAltitudeTime.IsZero() {
		altDrop := previousAltitude - t.AltitudeM
		timeDiff := t.Timestamp.Sub(previousAltitudeTime)
		if altDrop > 50 && timeDiff < 2*time.Second {
			alerts = append(alerts, telemetry.Alert{
				ID:        telemetry.GenerateAlertID(),
				VehicleID: t.VehicleID,
				Type:      "altitude_drop",
				Message:   fmt.Sprintf("Unexpected altitude drop: %.1fm in %.1fs", altDrop, timeDiff.Seconds()),
				Timestamp: time.Now(),
			})
		}
	}

	return alerts
}
