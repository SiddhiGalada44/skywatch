# skywatch

A lightweight, production-quality backend service for ingesting and monitoring real-time telemetry from unmanned aerial systems (UAS/drones). Simulates ground station functionality with alerting, state management, and REST API exposure.

## Overview

This project demonstrates a complete telemetry pipeline using Go's concurrency primitives (goroutines + channels) to handle high-throughput UDP ingestion, in-memory state management, and real-time alerting. It's designed for low-latency drone communications with graceful shutdown and structured logging.

## Architecture

```
UDP Ingestion (Port 8080)
    ↓
Parser → Telemetry Channel
    ↓
Processor Goroutine
    ↓
State Store (In-Memory)
    ↓
Alerting Engine
    ↓
REST API (Port 8081)
```

- **Ingestion**: UDP listener parses JSON telemetry messages
- **Processing**: Single goroutine processes messages, updates vehicle state, triggers alerts
- **State**: Thread-safe in-memory store of vehicle positions and alerts
- **Alerting**: Checks for low battery, altitude anomalies, and lost links
- **API**: HTTP endpoints for querying vehicles and alerts

## How to Run

### Prerequisites
- Go 1.21+

### Build
```bash
go build -o server ./cmd/server
go build -o simulator ./cmd/simulator
```

### Run Server
```bash
./server
```
Starts UDP listener on `:8080` and HTTP API on `:8081` by default. Override with environment variables:
```bash
UDP_ADDR=:9090 HTTP_ADDR=:9091 ./server
```

### Run Tests
```bash
go test ./...
```

### Run Simulator
```bash
./simulator
```
Simulates 3 drones sending telemetry to localhost:8080.

### Test API
```bash
curl http://localhost:8081/health
curl http://localhost:8081/vehicles
curl http://localhost:8081/alerts
```

## Design Decisions

### UDP for Ingestion
- **Why UDP?** Drones often operate in environments with intermittent connectivity where TCP's reliability overhead is counterproductive. UDP provides lower latency and simpler fire-and-forget messaging.
- **Tradeoff**: Potential message loss, mitigated by frequent telemetry sends (1Hz) and in-memory state that can tolerate gaps.

### In-Memory State Store
- **Why in-memory?** For this demo, persistence isn't needed. Real deployment would add Redis/PostgreSQL for durability.
- **Concurrency**: Uses RWMutex for thread-safe access. Single writer (processor goroutine) minimizes contention.

### Concurrency Model
- **Goroutines + Channels**: Processor uses a buffered channel to decouple ingestion from processing. Lost link checker runs on a ticker.
- **Why not worker pool?** Single processor suffices for expected load (~100 vehicles). Scales horizontally by running multiple instances.

### Alerting Logic
- **Per-Message Checks**: Battery and altitude alerts checked on each telemetry receipt.
- **Periodic Lost Link**: Separate goroutine checks for stale vehicles every second to avoid false positives from network jitter.

### No External Dependencies
- **Pure stdlib**: Keeps binary small and deployment simple. slog for structured JSON logging.

## What I'd Add with More Time

- **Persistence**: PostgreSQL for historical telemetry and alerts with time-series optimization.
- **WebSocket Live Feed**: Real-time updates for connected clients (e.g., ground station UI).
- **Kalman Filter**: Position smoothing and prediction for better anomaly detection.
- **Authentication**: API key validation for telemetry sources.
- **Metrics**: Prometheus integration for monitoring ingestion rates and alert frequencies.
- **Configuration**: YAML/JSON config file support for full threshold and timeout tuning (ports are already configurable via env vars).
- **Docker**: Containerized deployment with docker-compose for server + simulator.