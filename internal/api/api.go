package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"skywatch/internal/state"
)

// Server serves the REST API.
type Server struct {
	store  *state.Store
	router *http.ServeMux
}

// NewServer creates a new API server.
func NewServer(store *state.Store) *Server {
	s := &Server{
		store:  store,
		router: http.NewServeMux(),
	}
	s.routes()
	return s
}

// routes sets up the HTTP routes.
func (s *Server) routes() {
	s.router.HandleFunc("/vehicles", s.handleVehicles)
	s.router.HandleFunc("/vehicles/", s.handleVehicle)
	s.router.HandleFunc("/alerts", s.handleAlerts)
	s.router.HandleFunc("/health", s.handleHealth)
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// handleVehicles handles GET /vehicles.
func (s *Server) handleVehicles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	vehicles := s.store.GetAllVehicles()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(vehicles); err != nil {
		slog.Error("Failed to encode vehicles response", "error", err)
	}
}

// handleVehicle handles GET /vehicles/{id}.
func (s *Server) handleVehicle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/vehicles/")
	if id == "" {
		http.Error(w, "Missing vehicle ID", http.StatusBadRequest)
		return
	}
	t, exists := s.store.GetVehicle(id)
	if !exists {
		http.Error(w, "Vehicle not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(t); err != nil {
		slog.Error("Failed to encode vehicle response", "error", err)
	}
}

// handleAlerts handles GET /alerts.
func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	alerts := s.store.GetRecentAlerts()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(alerts); err != nil {
		slog.Error("Failed to encode alerts response", "error", err)
	}
}

// handleHealth handles GET /health.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	m := s.store.GetMetrics()
	resp := map[string]any{
		"status":         "ok",
		"vehicles":       m.Vehicles,
		"messages_total": m.MessagesTotal,
		"alerts_total":   m.AlertsTotal,
		"dropped_total":  m.DroppedTotal,
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("Failed to encode health response", "error", err)
	}
}
