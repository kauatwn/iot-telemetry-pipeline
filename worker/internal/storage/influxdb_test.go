package storage

import (
	"context"
	"testing"
	"worker/internal/telemetry"
)

type MockWriter struct {
	SavedPayloads []telemetry.Payload
	CloseCalled   bool
}

func (m *MockWriter) Save(ctx context.Context, p telemetry.Payload) error {
	m.SavedPayloads = append(m.SavedPayloads, p)
	return nil
}

func (m *MockWriter) Close() error {
	m.CloseCalled = true
	return nil
}

func TestMockWriterImplementsWriter(t *testing.T) {
	var _ Writer = (*MockWriter)(nil)

	mock := &MockWriter{}
	temp := 22.5
	payload := telemetry.Payload{
		DeviceID:    "device-test",
		Sensor:      "ds18b20",
		Temperature: &temp,
		Unit:        "celsius",
		UptimeMs:    5000,
		Status:      "OK",
	}

	if err := mock.Save(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error saving: %v", err)
	}

	if len(mock.SavedPayloads) != 1 {
		t.Fatalf("expected 1 saved payload, got: %d", len(mock.SavedPayloads))
	}

	if err := mock.Close(); err != nil {
		t.Fatalf("error closing mock: %v", err)
	}

	if !mock.CloseCalled {
		t.Error("expected CloseCalled to be true")
	}
}
