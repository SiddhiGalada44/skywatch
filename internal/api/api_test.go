package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"skywatch/internal/state"
	"skywatch/internal/telemetry"
)

func setupServer() *Server {
	store := state.NewStore()
	store.UpdateTelemetry(telemetry.Telemetry{
		VehicleID:   "drone-1",
		Timestamp:   time.Now(),
		Lat:         37.7749,
		Lon:         -122.4194,
		AltitudeM:   100,
		SpeedMPS:    10,
		BatteryPct:  80,
		FlightPhase: "cruise",
	})
	store.AddAlert(telemetry.Alert{
		ID:        "alert-1",
		VehicleID: "drone-1",
		Type:      "low_battery",
		Message:   "Battery low: 15.0%",
		Timestamp: time.Now(),
	})
	return NewServer(store)
}

func TestHandleVehicles(t *testing.T) {
	srv := setupServer()

	req := httptest.NewRequest(http.MethodGet, "/vehicles", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var result map[string]telemetry.Telemetry
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if _, ok := result["drone-1"]; !ok {
		t.Error("expected drone-1 in response")
	}
}

func TestHandleVehicles_MethodNotAllowed(t *testing.T) {
	srv := setupServer()

	req := httptest.NewRequest(http.MethodPost, "/vehicles", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleVehicle_Found(t *testing.T) {
	srv := setupServer()

	req := httptest.NewRequest(http.MethodGet, "/vehicles/drone-1", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var result telemetry.Telemetry
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.VehicleID != "drone-1" {
		t.Errorf("expected drone-1, got %s", result.VehicleID)
	}
}

func TestHandleVehicle_NotFound(t *testing.T) {
	srv := setupServer()

	req := httptest.NewRequest(http.MethodGet, "/vehicles/ghost", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandleAlerts(t *testing.T) {
	srv := setupServer()

	req := httptest.NewRequest(http.MethodGet, "/alerts", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var alerts []telemetry.Alert
	if err := json.NewDecoder(w.Body).Decode(&alerts); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(alerts) != 1 {
		t.Errorf("expected 1 alert, got %d", len(alerts))
	}
	if alerts[0].Type != "low_battery" {
		t.Errorf("expected low_battery alert, got %s", alerts[0].Type)
	}
}

func TestHandleHealth(t *testing.T) {
	srv := setupServer()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var result map[string]any
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("expected status ok, got %v", result["status"])
	}
	if _, ok := result["messages_total"]; !ok {
		t.Error("expected messages_total in health response")
	}
	if _, ok := result["dropped_total"]; !ok {
		t.Error("expected dropped_total in health response")
	}
}

func TestHandleHealth_MethodNotAllowed(t *testing.T) {
	srv := setupServer()

	req := httptest.NewRequest(http.MethodDelete, "/health", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}
