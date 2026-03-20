package state

import (
	"testing"
	"time"

	"skywatch/internal/telemetry"
)

func makeTelemetry(vehicleID string, altitude, battery float64, phase string) telemetry.Telemetry {
	return telemetry.Telemetry{
		VehicleID:   vehicleID,
		Timestamp:   time.Now(),
		Lat:         37.7749,
		Lon:         -122.4194,
		AltitudeM:   altitude,
		SpeedMPS:    10,
		BatteryPct:  battery,
		FlightPhase: phase,
	}
}

func TestStore_UpdateAndGetVehicle(t *testing.T) {
	store := NewStore()

	_, exists := store.GetVehicle("drone-1")
	if exists {
		t.Fatal("expected no vehicle before any telemetry")
	}

	tel := makeTelemetry("drone-1", 100, 80, "cruise")
	store.UpdateTelemetry(tel)

	got, exists := store.GetVehicle("drone-1")
	if !exists {
		t.Fatal("expected vehicle to exist after telemetry update")
	}
	if got.VehicleID != "drone-1" {
		t.Errorf("expected vehicle_id drone-1, got %s", got.VehicleID)
	}
	if got.AltitudeM != 100 {
		t.Errorf("expected altitude 100, got %f", got.AltitudeM)
	}
}

func TestStore_GetAllVehicles(t *testing.T) {
	store := NewStore()

	store.UpdateTelemetry(makeTelemetry("drone-1", 100, 80, "cruise"))
	store.UpdateTelemetry(makeTelemetry("drone-2", 50, 60, "loiter"))

	vehicles := store.GetAllVehicles()
	if len(vehicles) != 2 {
		t.Errorf("expected 2 vehicles, got %d", len(vehicles))
	}
	if _, ok := vehicles["drone-1"]; !ok {
		t.Error("expected drone-1 in vehicles")
	}
	if _, ok := vehicles["drone-2"]; !ok {
		t.Error("expected drone-2 in vehicles")
	}
}

func TestStore_PreviousAltitudeTracked(t *testing.T) {
	store := NewStore()

	store.UpdateTelemetry(makeTelemetry("drone-1", 100, 80, "cruise"))
	store.UpdateTelemetry(makeTelemetry("drone-1", 60, 80, "descent"))

	prev := store.GetPreviousAltitude("drone-1")
	if prev != 100 {
		t.Errorf("expected previous altitude 100, got %f", prev)
	}
}

func TestStore_AddAndGetAlerts(t *testing.T) {
	store := NewStore()

	alert := telemetry.Alert{
		ID:        "alert-1",
		VehicleID: "drone-1",
		Type:      "low_battery",
		Message:   "Battery low: 15.0%",
		Timestamp: time.Now(),
	}
	store.AddAlert(alert)

	alerts := store.GetRecentAlerts()
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}
	if alerts[0].Type != "low_battery" {
		t.Errorf("expected low_battery alert, got %s", alerts[0].Type)
	}
}

func TestStore_AlertsCappedAt50(t *testing.T) {
	store := NewStore()

	for i := 0; i < 60; i++ {
		store.AddAlert(telemetry.Alert{
			ID:        telemetry.GenerateAlertID(),
			VehicleID: "drone-1",
			Type:      "low_battery",
			Timestamp: time.Now(),
		})
	}

	alerts := store.GetRecentAlerts()
	if len(alerts) != 50 {
		t.Errorf("expected alerts capped at 50, got %d", len(alerts))
	}
}

func TestStore_GetRecentAlertsReturnsCopy(t *testing.T) {
	store := NewStore()
	store.AddAlert(telemetry.Alert{ID: "1", VehicleID: "drone-1", Type: "low_battery", Timestamp: time.Now()})

	alerts := store.GetRecentAlerts()
	alerts[0].Type = "tampered"

	fresh := store.GetRecentAlerts()
	if fresh[0].Type == "tampered" {
		t.Error("GetRecentAlerts should return a copy, not a reference to internal state")
	}
}

func TestStore_CheckLostLinks(t *testing.T) {
	store := NewStore()

	// Manually insert a vehicle with an old LastSeen time
	store.mu.Lock()
	store.vehicles["drone-ghost"] = &VehicleState{
		LastSeen:        time.Now().Add(-10 * time.Second),
		LostLinkAlerted: false,
	}
	store.mu.Unlock()

	alerts := store.CheckLostLinks()
	if len(alerts) != 1 {
		t.Fatalf("expected 1 lost_link alert, got %d", len(alerts))
	}
	if alerts[0].Type != "lost_link" {
		t.Errorf("expected lost_link alert, got %s", alerts[0].Type)
	}
}

func TestStore_CheckLostLinks_NoDoubleAlert(t *testing.T) {
	store := NewStore()

	store.mu.Lock()
	store.vehicles["drone-ghost"] = &VehicleState{
		LastSeen:        time.Now().Add(-10 * time.Second),
		LostLinkAlerted: false,
	}
	store.mu.Unlock()

	first := store.CheckLostLinks()
	second := store.CheckLostLinks()

	if len(first) != 1 {
		t.Fatalf("expected 1 alert on first check, got %d", len(first))
	}
	if len(second) != 0 {
		t.Errorf("expected 0 alerts on second check (no duplicates), got %d", len(second))
	}
}

func TestStore_LowBatteryDeduplication(t *testing.T) {
	store := NewStore()
	store.UpdateTelemetry(makeTelemetry("drone-1", 100, 80, "cruise"))

	// First mark should succeed
	if !store.MarkLowBatteryAlerted("drone-1") {
		t.Error("expected first MarkLowBatteryAlerted to return true")
	}
	// Second mark should be blocked
	if store.MarkLowBatteryAlerted("drone-1") {
		t.Error("expected second MarkLowBatteryAlerted to return false (dedup)")
	}

	// After battery recovers (>=20%), flag resets
	store.UpdateTelemetry(makeTelemetry("drone-1", 100, 50, "cruise"))
	if !store.MarkLowBatteryAlerted("drone-1") {
		t.Error("expected MarkLowBatteryAlerted to return true after battery recovery")
	}
}
