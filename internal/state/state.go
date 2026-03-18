package state

import (
	"sync"
	"time"

	"skywatch/internal/telemetry"
)

// VehicleState holds the current state of a vehicle.
type VehicleState struct {
	LastTelemetry        telemetry.Telemetry
	PreviousAltitude     float64
	PreviousAltitudeTime time.Time
	LastSeen             time.Time
	LostLinkAlerted      bool
}

// Store manages the in-memory state of all vehicles and alerts.
type Store struct {
	mu       sync.RWMutex
	vehicles map[string]*VehicleState
	alerts   []telemetry.Alert
}

// NewStore creates a new state store.
func NewStore() *Store {
	return &Store{
		vehicles: make(map[string]*VehicleState),
		alerts:   make([]telemetry.Alert, 0),
	}
}

// UpdateTelemetry updates the state with new telemetry and checks for alerts.
func (s *Store) UpdateTelemetry(t telemetry.Telemetry) {
	s.mu.Lock()
	defer s.mu.Unlock()

	vs, exists := s.vehicles[t.VehicleID]
	if !exists {
		vs = &VehicleState{}
		s.vehicles[t.VehicleID] = vs
	}

	// Update previous altitude if exists
	if exists {
		vs.PreviousAltitude = vs.LastTelemetry.AltitudeM
		vs.PreviousAltitudeTime = vs.LastTelemetry.Timestamp
	}

	vs.LastTelemetry = t
	vs.LastSeen = time.Now()   // Use server time for reliability
	vs.LostLinkAlerted = false // Reset lost link alert on new telemetry
}

// GetVehicle returns the state of a vehicle.
func (s *Store) GetVehicle(id string) (telemetry.Telemetry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	vs, exists := s.vehicles[id]
	if !exists {
		return telemetry.Telemetry{}, false
	}
	return vs.LastTelemetry, true
}

// GetAllVehicles returns all vehicles' last telemetry.
func (s *Store) GetAllVehicles() map[string]telemetry.Telemetry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]telemetry.Telemetry)
	for id, vs := range s.vehicles {
		result[id] = vs.LastTelemetry
	}
	return result
}

// AddAlert adds a new alert.
func (s *Store) AddAlert(alert telemetry.Alert) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alerts = append(s.alerts, alert)
	// Keep only last 50 alerts
	if len(s.alerts) > 50 {
		s.alerts = s.alerts[len(s.alerts)-50:]
	}
}

// GetRecentAlerts returns the last 50 alerts.
func (s *Store) GetRecentAlerts() []telemetry.Alert {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]telemetry.Alert(nil), s.alerts...) // Copy
}

// CheckLostLinks checks for vehicles that haven't sent telemetry in >5 seconds.
func (s *Store) CheckLostLinks() []telemetry.Alert {
	s.mu.Lock()
	defer s.mu.Unlock()
	var alerts []telemetry.Alert
	now := time.Now()
	for id, vs := range s.vehicles {
		if !vs.LostLinkAlerted && now.Sub(vs.LastSeen) > 5*time.Second {
			alert := telemetry.Alert{
				ID:        telemetry.GenerateAlertID(),
				VehicleID: id,
				Type:      "lost_link",
				Message:   "Lost telemetry link",
				Timestamp: now,
			}
			s.alerts = append(s.alerts, alert)
			if len(s.alerts) > 50 {
				s.alerts = s.alerts[len(s.alerts)-50:]
			}
			vs.LostLinkAlerted = true
			alerts = append(alerts, alert)
		}
	}
	return alerts
}

// GetPreviousAltitude returns the previous altitude for a vehicle.
func (s *Store) GetPreviousAltitude(id string) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	vs, exists := s.vehicles[id]
	if !exists {
		return 0
	}
	return vs.PreviousAltitude
}

// GetPreviousAltitudeTime returns the previous altitude timestamp for a vehicle.
func (s *Store) GetPreviousAltitudeTime(id string) time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	vs, exists := s.vehicles[id]
	if !exists {
		return time.Time{}
	}
	return vs.PreviousAltitudeTime
}

