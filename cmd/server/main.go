package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"skywatch/internal/alerting"
	"skywatch/internal/api"
	"skywatch/internal/ingestion"
	"skywatch/internal/state"
	"skywatch/internal/telemetry"
)

func main() {
	// Setup structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{}))
	slog.SetDefault(logger)

	slog.Info("Starting skywatch server")

	// Config from environment with defaults
	udpAddr := os.Getenv("UDP_ADDR")
	if udpAddr == "" {
		udpAddr = ":8080"
	}
	httpAddr := os.Getenv("HTTP_ADDR")
	if httpAddr == "" {
		httpAddr = ":8081"
	}

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize components
	store := state.NewStore()
	teleCh := make(chan telemetry.Telemetry, 100) // Buffered channel for ingestion

	listener := ingestion.NewListener(ctx, udpAddr, teleCh, store)
	apiServer := api.NewServer(store)

	// Signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start ingestion listener
	go func() {
		slog.Info("Starting UDP listener", "addr", udpAddr)
		if err := listener.Start(); err != nil {
			slog.Error("UDP listener failed", "error", err)
			cancel()
		}
	}()

	// Start HTTP server
	httpServer := &http.Server{
		Addr:    httpAddr,
		Handler: apiServer,
	}
	go func() {
		slog.Info("Starting HTTP server", "addr", httpAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server failed", "error", err)
			cancel()
		}
	}()

	// Processor goroutine
	go func() {
		for {
			select {
			case t := <-teleCh:
				slog.Debug("Received telemetry", "vehicle_id", t.VehicleID, "phase", t.FlightPhase)
				store.UpdateTelemetry(t)
				alerts := alerting.CheckAlerts(t, store.GetPreviousAltitude(t.VehicleID), store.GetPreviousAltitudeTime(t.VehicleID))
				for _, alert := range alerts {
					if alert.Type == "low_battery" && !store.MarkLowBatteryAlerted(t.VehicleID) {
						continue // Already alerted, skip
					}
					store.AddAlert(alert)
					slog.Info("Alert triggered", "type", alert.Type, "vehicle_id", alert.VehicleID, "message", alert.Message)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Lost link checker
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				alerts := store.CheckLostLinks()
				for _, alert := range alerts {
					slog.Info("Alert triggered", "type", alert.Type, "vehicle_id", alert.VehicleID, "message", alert.Message)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for shutdown signal
	<-sigCh
	slog.Info("Shutting down gracefully")

	// Shutdown HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP server shutdown failed", "error", err)
	}

	cancel() // Cancel context to stop goroutines

	slog.Info("Shutdown complete")
}
