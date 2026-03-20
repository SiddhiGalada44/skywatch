package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"skywatch/internal/telemetry"
)

func main() {
	drones := flag.Int("drones", 100, "number of simulated drones")
	rate := flag.Int("rate", 10, "messages per second per drone")
	duration := flag.Duration("duration", 30*time.Second, "how long to run")
	addr := flag.String("addr", "localhost:8080", "server UDP address")
	flag.Parse()

	fmt.Printf("Stress test: %d drones x %d msg/s for %s → %s\n", *drones, *rate, *duration, *addr)
	fmt.Printf("Total target: %d msg/s\n\n", *drones**rate)

	conn, err := net.Dial("udp", *addr)
	if err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		return
	}
	defer conn.Close()

	var sent atomic.Int64
	var failed atomic.Int64

	stop := make(chan struct{})
	time.AfterFunc(*duration, func() { close(stop) })

	interval := time.Second / time.Duration(*rate)

	var wg sync.WaitGroup
	for i := 0; i < *drones; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			vehicleID := fmt.Sprintf("stress-drone-%04d", id)
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			battery := 100.0
			altitude := 100.0
			phases := []string{"takeoff", "cruise", "loiter", "descent"}
			for {
				select {
				case <-stop:
					return
				case <-ticker.C:
					battery -= rand.Float64() * 0.1
					if battery < 0 {
						battery = 0
					}
					t := telemetry.Telemetry{
						VehicleID:   vehicleID,
						Timestamp:   time.Now(),
						Lat:         37.7749 + rand.Float64()*0.1,
						Lon:         -122.4194 + rand.Float64()*0.1,
						AltitudeM:   altitude,
						SpeedMPS:    rand.Float64() * 20,
						BatteryPct:  battery,
						FlightPhase: phases[rand.Intn(len(phases))],
					}
					data, _ := json.Marshal(t)
					if _, err := conn.Write(data); err != nil {
						failed.Add(1)
					} else {
						sent.Add(1)
					}
				}
			}
		}(i)
	}

	// Reporter: print throughput every second
	reportTicker := time.NewTicker(1 * time.Second)
	defer reportTicker.Stop()
	lastSent := int64(0)
	start := time.Now()
	for {
		select {
		case <-stop:
			wg.Wait()
			totalSent := sent.Load()
			totalFailed := failed.Load()
			elapsed := time.Since(start).Seconds()
			fmt.Printf("\n--- Results ---\n")
			fmt.Printf("Duration:     %.1fs\n", elapsed)
			fmt.Printf("Sent:         %d messages\n", totalSent)
			fmt.Printf("Failed:       %d messages\n", totalFailed)
			fmt.Printf("Avg rate:     %.0f msg/s\n", float64(totalSent)/elapsed)
			return
		case <-reportTicker.C:
			now := sent.Load()
			rate := now - lastSent
			lastSent = now
			fmt.Printf("[%5.1fs] sent/s: %-6d  total: %d  failed: %d\n",
				time.Since(start).Seconds(), rate, now, failed.Load())
		}
	}
}
