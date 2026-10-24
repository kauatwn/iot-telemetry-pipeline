package telemetry

import (
	"context"
	"log/slog"
	"time"
)

const writeTimeout = 5 * time.Second

// Writer define o contrato para persistência dos dados de telemetria.
type Writer interface {
	Save(ctx context.Context, p Payload) error
	Close() error
}

// Processor consome o canal de mensagens e as envia para o armazenamento.
type Processor struct {
	writer Writer
}

func NewProcessor(writer Writer) *Processor {
	return &Processor{writer: writer}
}

// Process executa o loop principal de consumo da fila.
func (p *Processor) Process(payloadChan <-chan Payload) {
	slog.Info("waiting for telemetry messages in pipeline...")
	for payload := range payloadChan {

		if payload.IsDisconnected() {
			slog.Warn("processing telemetry (sensor disconnected)",
				"device_id", payload.DeviceID,
				"sensor", payload.Sensor,
				"unit", payload.Unit,
				"status", payload.Status,
			)
		} else {
			slog.Info("processing telemetry",
				"device_id", payload.DeviceID,
				"sensor", payload.Sensor,
				"temperature", *payload.Temperature,
				"unit", payload.Unit,
				"status", payload.Status,
			)
		}

		ctx, cancel := context.WithTimeout(context.Background(), writeTimeout)
		if err := p.writer.Save(ctx, payload); err != nil {
			slog.Error("failed to save telemetry to InfluxDB",
				"error", err,
				"device_id", payload.DeviceID,
			)
		}
		cancel()
	}
	slog.Info("consumer goroutine finished draining channel")
}
