package telemetry

import (
	"encoding/json"
	"testing"
)

func TestPayloadUnmarshal_ValidTemperature(t *testing.T) {
	data := []byte(`{
		"device_id": "esp32-sensor-01",
		"sensor": "ds18b20",
		"temperature": 24.50,
		"unit": "celsius",
		"uptime_ms": 12000,
		"status": "OK"
	}`)

	var p Payload
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if p.DeviceID != "esp32-sensor-01" {
		t.Errorf("expected device_id 'esp32-sensor-01', got: %q", p.DeviceID)
	}
	if !p.HasTemperature() {
		t.Fatal("expected HasTemperature() to be true, got false")
	}
	if *p.Temperature != 24.50 {
		t.Errorf("expected temperature 24.50, got: %f", *p.Temperature)
	}
	if p.UptimeMs != 12000 {
		t.Errorf("expected uptime_ms 12000, got: %d", p.UptimeMs)
	}
	if p.Status != "OK" {
		t.Errorf("expected status 'OK', got: %q", p.Status)
	}
}

func TestPayloadUnmarshal_DisconnectedSensor(t *testing.T) {
	data := []byte(`{
		"device_id": "esp32-sensor-01",
		"sensor": "ds18b20",
		"temperature": null,
		"unit": "celsius",
		"uptime_ms": 14000,
		"status": "DISCONNECTED"
	}`)

	var p Payload
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if p.HasTemperature() {
		t.Fatal("expected HasTemperature() to be false for disconnected sensor, got true")
	}
	if p.Status != "DISCONNECTED" {
		t.Errorf("expected status 'DISCONNECTED', got: %q", p.Status)
	}
}
