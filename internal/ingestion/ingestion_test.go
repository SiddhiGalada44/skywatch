package ingestion

import (
	"testing"
	"time"

	"skywatch/internal/telemetry"
)

func TestParser_Parse(t *testing.T) {
	p := NewParser()

	tests := []struct {
		name    string
		data    string
		want    telemetry.Telemetry
		wantErr bool
	}{
		{
			name: "valid telemetry",
			data: `{
				"vehicle_id": "drone-1",
				"timestamp": "2023-01-01T00:00:00Z",
				"lat": 37.7749,
				"lon": -122.4194,
				"altitude_m": 100,
				"speed_mps": 10,
				"battery_pct": 80,
				"flight_phase": "cruise"
			}`,
			want: telemetry.Telemetry{
				VehicleID:   "drone-1",
				Timestamp:   time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				Lat:         37.7749,
				Lon:         -122.4194,
				AltitudeM:   100,
				SpeedMPS:    10,
				BatteryPct:  80,
				FlightPhase: "cruise",
			},
			wantErr: false,
		},
		{
			name:    "invalid json",
			data:    `{invalid}`,
			wantErr: true,
		},
		{
			name: "missing vehicle_id",
			data: `{
				"timestamp": "2023-01-01T00:00:00Z",
				"lat": 37.7749,
				"lon": -122.4194,
				"altitude_m": 100,
				"speed_mps": 10,
				"battery_pct": 80,
				"flight_phase": "cruise"
			}`,
			wantErr: true,
		},
		{
			name: "invalid battery_pct",
			data: `{
				"vehicle_id": "drone-1",
				"timestamp": "2023-01-01T00:00:00Z",
				"lat": 37.7749,
				"lon": -122.4194,
				"altitude_m": 100,
				"speed_mps": 10,
				"battery_pct": 150,
				"flight_phase": "cruise"
			}`,
			wantErr: true,
		},
		{
			name: "invalid flight_phase",
			data: `{
				"vehicle_id": "drone-1",
				"timestamp": "2023-01-01T00:00:00Z",
				"lat": 37.7749,
				"lon": -122.4194,
				"altitude_m": 100,
				"speed_mps": 10,
				"battery_pct": 80,
				"flight_phase": "invalid"
			}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := p.Parse([]byte(tt.data))
			if (err != nil) != tt.wantErr {
				t.Errorf("Parser.Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Parser.Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkParser_Parse(b *testing.B) {
	p := &Parser{}
	data := []byte(`{
		"vehicle_id": "drone-1",
		"timestamp": "2023-01-01T00:00:00Z",
		"lat": 37.7749,
		"lon": -122.4194,
		"altitude_m": 100,
		"speed_mps": 10,
		"battery_pct": 80,
		"flight_phase": "cruise"
	}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = p.Parse(data)
	}
}
