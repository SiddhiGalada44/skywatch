package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"skywatch/internal/telemetry"
)

// Parser parses incoming telemetry data.
type Parser struct {
	pool sync.Pool
}

// NewParser creates a new parser with object pooling.
func NewParser() *Parser {
	return &Parser{
		pool: sync.Pool{
			New: func() interface{} {
				return &telemetry.Telemetry{}
			},
		},
	}
}

// Parse parses JSON bytes into Telemetry.
func (p *Parser) Parse(data []byte) (telemetry.Telemetry, error) {
	tPtr := p.pool.Get()
	var t *telemetry.Telemetry
	if tPtr == nil {
		t = &telemetry.Telemetry{}
	} else {
		t = tPtr.(*telemetry.Telemetry)
	}
	*t = telemetry.Telemetry{} // Reset to zero values
	err := json.Unmarshal(data, t)
	if err != nil {
		p.pool.Put(t) // Return to pool even on error
		var zero telemetry.Telemetry
		return zero, err
	}
	// Basic validation
	if t.VehicleID == "" {
		p.pool.Put(t)
		return telemetry.Telemetry{}, fmt.Errorf("missing vehicle_id")
	}
	if t.BatteryPct < 0 || t.BatteryPct > 100 {
		p.pool.Put(t)
		return telemetry.Telemetry{}, fmt.Errorf("invalid battery_pct: %f", t.BatteryPct)
	}
	if t.AltitudeM < 0 {
		p.pool.Put(t)
		return telemetry.Telemetry{}, fmt.Errorf("invalid altitude_m: %f", t.AltitudeM)
	}
	if t.SpeedMPS < 0 {
		p.pool.Put(t)
		return telemetry.Telemetry{}, fmt.Errorf("invalid speed_mps: %f", t.SpeedMPS)
	}
	if t.Lat < -90 || t.Lat > 90 {
		p.pool.Put(t)
		return telemetry.Telemetry{}, fmt.Errorf("invalid lat: %f", t.Lat)
	}
	if t.Lon < -180 || t.Lon > 180 {
		p.pool.Put(t)
		return telemetry.Telemetry{}, fmt.Errorf("invalid lon: %f", t.Lon)
	}
	validPhases := map[string]bool{"takeoff": true, "cruise": true, "descent": true, "loiter": true}
	if !validPhases[t.FlightPhase] {
		p.pool.Put(t)
		return telemetry.Telemetry{}, fmt.Errorf("invalid flight_phase: %s", t.FlightPhase)
	}
	if t.Timestamp.IsZero() {
		t.Timestamp = time.Now() // Fallback
	}
	// Copy the telemetry to return, return the pointer to pool
	result := *t
	p.pool.Put(t)
	return result, nil
}

// Dropper is implemented by anything that can count dropped messages.
type Dropper interface {
	IncDropped()
}

// Listener listens for UDP telemetry messages.
type Listener struct {
	addr    string
	parser  *Parser
	teleCh  chan<- telemetry.Telemetry
	ctx     context.Context
	dropper Dropper
}

// NewListener creates a new UDP listener.
func NewListener(ctx context.Context, addr string, teleCh chan<- telemetry.Telemetry, dropper Dropper) *Listener {
	return &Listener{
		addr:    addr,
		parser:  NewParser(),
		teleCh:  teleCh,
		ctx:     ctx,
		dropper: dropper,
	}
}

// Start starts the UDP listener.
func (l *Listener) Start() error {
	conn, err := net.ListenPacket("udp", l.addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	buf := make([]byte, 1024)
	for {
		select {
		case <-l.ctx.Done():
			return nil
		default:
		}
		conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		n, _, err := conn.ReadFrom(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			return err
		}
		t, err := l.parser.Parse(buf[:n])
		if err != nil {
			// Log error, but continue
			continue
		}
		select {
		case l.teleCh <- t:
		case <-l.ctx.Done():
			return nil
		default:
			slog.Warn("Telemetry channel full, dropping message", "vehicle_id", t.VehicleID)
			if l.dropper != nil {
				l.dropper.IncDropped()
			}
		}
	}
}
