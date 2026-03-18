package alerting

import (
	"testing"
	"time"

	"skywatch/internal/telemetry"
)

func TestCheckAlerts(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name               string
		t                  telemetry.Telemetry
		previousAltitude   float64
		previousTime       time.Time
		expectedAlertTypes []string
	}{
		{
			name: "low battery",
			t: telemetry.Telemetry{
				VehicleID:  "drone-1",
				Timestamp:  now,
				BatteryPct: 15,
			},
			expectedAlertTypes: []string{"low_battery"},
		},
		{
			name: "altitude drop",
			t: telemetry.Telemetry{
				VehicleID:  "drone-1",
				Timestamp:  now,
				AltitudeM:  40,
				BatteryPct: 50,
			},
			previousAltitude:   100,
			previousTime:       now.Add(-1 * time.Second),
			expectedAlertTypes: []string{"altitude_drop"},
		},
		{
			name: "no alerts",
			t: telemetry.Telemetry{
				VehicleID:  "drone-1",
				Timestamp:  now,
				BatteryPct: 50,
				AltitudeM:  100,
			},
			previousAltitude:   100,
			previousTime:       now.Add(-1 * time.Second),
			expectedAlertTypes: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alerts := CheckAlerts(tt.t, tt.previousAltitude, tt.previousTime)
			if len(alerts) != len(tt.expectedAlertTypes) {
				t.Errorf("Expected %d alerts, got %d", len(tt.expectedAlertTypes), len(alerts))
			}
			for i, alert := range alerts {
				if alert.Type != tt.expectedAlertTypes[i] {
					t.Errorf("Expected alert type %s, got %s", tt.expectedAlertTypes[i], alert.Type)
				}
			}
		})
	}
}
