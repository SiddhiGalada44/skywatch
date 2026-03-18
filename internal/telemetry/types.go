package telemetry

import (
	"fmt"
	"time"
)

// Telemetry represents a single telemetry message from a UAS vehicle.
type Telemetry struct {
	VehicleID   string    `json:"vehicle_id"`
	Timestamp   time.Time `json:"timestamp"`
	Lat         float64   `json:"lat"`
	Lon         float64   `json:"lon"`
	AltitudeM   float64   `json:"altitude_m"`
	SpeedMPS    float64   `json:"speed_mps"`
	BatteryPct  float64   `json:"battery_pct"`
	FlightPhase string    `json:"flight_phase"` // takeoff, cruise, descent, loiter
}

// Alert represents an alert triggered by the system.
type Alert struct {
	ID        string    `json:"id"`
	VehicleID string    `json:"vehicle_id"`
	Type      string    `json:"type"` // low_battery, altitude_drop, lost_link
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// GenerateAlertID generates a unique ID for alerts using nanosecond precision.
func GenerateAlertID() string {
	now := time.Now()
	return fmt.Sprintf("%s-%d", now.Format("20060102150405"), now.UnixNano())
}
