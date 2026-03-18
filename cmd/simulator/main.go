package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net"
	"time"

	"skywatch/internal/telemetry"
)

func main() {
	conn, err := net.Dial("udp", "localhost:8080")
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	// Simulate 3 vehicles
	go simulateVehicle(conn, "drone-1", normalFlight)
	go simulateVehicle(conn, "drone-2", lowBatteryFlight)
	go simulateVehicle(conn, "drone-3", lostLinkFlight)

	// Wait forever
	select {}
}

func simulateVehicle(conn net.Conn, vehicleID string, flightFunc func(string, net.Conn)) {
	flightFunc(vehicleID, conn)
}

func normalFlight(vehicleID string, conn net.Conn) {
	phases := []string{"takeoff", "cruise", "loiter", "descent"}
	durations := []time.Duration{10 * time.Second, 20 * time.Second, 15 * time.Second, 10 * time.Second}

	startTime := time.Now()
	phaseIndex := 0
	altitude := 0.0
	battery := 100.0

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			elapsed := time.Since(startTime)
			if elapsed > durations[phaseIndex] && phaseIndex < len(phases)-1 {
				phaseIndex++
				startTime = time.Now()
			}

			phase := phases[phaseIndex]

			// Update altitude and battery
			switch phase {
			case "takeoff":
				altitude += 5
				battery -= 0.5
			case "cruise":
				altitude = 100
				battery -= 0.2
			case "loiter":
				altitude = 100
				battery -= 0.1
			case "descent":
				altitude -= 5
				battery -= 0.3
			}

			if altitude < 0 {
				altitude = 0
			}
			if battery < 0 {
				battery = 0
			}

			t := telemetry.Telemetry{
				VehicleID:   vehicleID,
				Timestamp:   time.Now(),
				Lat:         37.7749 + math.Sin(float64(time.Now().Unix()))*0.01,
				Lon:         -122.4194 + math.Cos(float64(time.Now().Unix()))*0.01,
				AltitudeM:   altitude,
				SpeedMPS:    10,
				BatteryPct:  battery,
				FlightPhase: phase,
			}

			sendTelemetry(conn, t)
		}
	}
}

func lowBatteryFlight(vehicleID string, conn net.Conn) {
	totalStart := time.Now()
	// Similar to normal, but set battery low at some point
	phases := []string{"takeoff", "cruise", "loiter", "descent"}
	durations := []time.Duration{10 * time.Second, 20 * time.Second, 15 * time.Second, 10 * time.Second}

	startTime := time.Now()
	phaseIndex := 0
	altitude := 0.0
	battery := 100.0

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			elapsed := time.Since(startTime)
			if elapsed > durations[phaseIndex] && phaseIndex < len(phases)-1 {
				phaseIndex++
				startTime = time.Now()
			}

			phase := phases[phaseIndex]

			// Update altitude and battery
			switch phase {
			case "takeoff":
				altitude += 5
				battery -= 0.5
			case "cruise":
				altitude = 100
				if time.Since(totalStart) > 30*time.Second {
					battery = 15
				} else {
					battery -= 0.2
				}
			case "loiter":
				altitude = 100
				battery -= 0.1
			case "descent":
				altitude -= 5
				battery -= 0.3
			}

			if altitude < 0 {
				altitude = 0
			}
			if battery < 0 {
				battery = 0
			}

			t := telemetry.Telemetry{
				VehicleID:   vehicleID,
				Timestamp:   time.Now(),
				Lat:         37.7749 + math.Sin(float64(time.Now().Unix()))*0.01,
				Lon:         -122.4194 + math.Cos(float64(time.Now().Unix()))*0.01,
				AltitudeM:   altitude,
				SpeedMPS:    10,
				BatteryPct:  battery,
				FlightPhase: phase,
			}

			sendTelemetry(conn, t)
		}
	}
}

func lostLinkFlight(vehicleID string, conn net.Conn) {
	totalStart := time.Now()
	// Similar, but stop sending after some time to simulate lost link
	phases := []string{"takeoff", "cruise", "loiter", "descent"}
	durations := []time.Duration{10 * time.Second, 20 * time.Second, 15 * time.Second, 10 * time.Second}

	startTime := time.Now()
	phaseIndex := 0
	altitude := 0.0
	battery := 100.0

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			totalElapsed := time.Since(totalStart)
			if totalElapsed > 45*time.Second {
				continue // Stop sending to simulate lost link
			}

			elapsed := time.Since(startTime)
			if elapsed > durations[phaseIndex] && phaseIndex < len(phases)-1 {
				phaseIndex++
				startTime = time.Now()
			}

			phase := phases[phaseIndex]

			// Update altitude and battery
			switch phase {
			case "takeoff":
				altitude += 5
				battery -= 0.5
			case "cruise":
				altitude = 100
				battery -= 0.2
			case "loiter":
				altitude = 100
				battery -= 0.1
			case "descent":
				altitude -= 5
				battery -= 0.3
			}

			if altitude < 0 {
				altitude = 0
			}
			if battery < 0 {
				battery = 0
			}

			t := telemetry.Telemetry{
				VehicleID:   vehicleID,
				Timestamp:   time.Now(),
				Lat:         37.7749 + math.Sin(float64(time.Now().Unix()))*0.01,
				Lon:         -122.4194 + math.Cos(float64(time.Now().Unix()))*0.01,
				AltitudeM:   altitude,
				SpeedMPS:    10,
				BatteryPct:  battery,
				FlightPhase: phase,
			}

			sendTelemetry(conn, t)
		}
	}
}

func sendTelemetry(conn net.Conn, t telemetry.Telemetry) {
	data, err := json.Marshal(t)
	if err != nil {
		fmt.Println("Error marshaling:", err)
		return
	}
	_, err = conn.Write(data)
	if err != nil {
		fmt.Println("Error sending:", err)
	}
}
