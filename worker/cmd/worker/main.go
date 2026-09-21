package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"worker/internal/broker"
	"worker/internal/config"
	"worker/internal/storage"
	"worker/internal/telemetry"
)

const (
	writeTimeout       = 5 * time.Second
	telemetryQueueSize = 100
)

func setupLogger(cfg config.LogConfig) {
	var level slog.Level
	switch strings.ToUpper(cfg.Level) {
	case "DEBUG":
		level = slog.LevelDebug
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if strings.ToLower(cfg.Format) == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(handler))
}

// processTelemetry consome as mensagens da esteira e persiste no storage com controle de timeout.
func processTelemetry(db storage.Writer, payloadChan <-chan telemetry.Payload) {
	slog.Info("waiting for telemetry messages in pipeline...")
	for payload := range payloadChan {
		logTelemetry(payload)

		writeCtx, cancel := context.WithTimeout(context.Background(), writeTimeout)
		if err := db.Save(writeCtx, payload); err != nil {
			slog.Error("failed to save telemetry to InfluxDB",
				"error", err,
				"device_id", payload.DeviceID,
			)
		}
		cancel()
	}
	slog.Info("consumer goroutine finished draining channel")
}

// logTelemetry emite o log estruturado da telemetria recebida de acordo com a integridade física do sensor.
func logTelemetry(p telemetry.Payload) {
	if p.HasTemperature() {
		slog.Info("processing telemetry",
			"device_id", p.DeviceID,
			"sensor", p.Sensor,
			"temperature", *p.Temperature,
			"unit", p.Unit,
			"status", p.Status,
		)
		return
	}

	slog.Warn("processing telemetry (sensor disconnected)",
		"device_id", p.DeviceID,
		"sensor", p.Sensor,
		"unit", p.Unit,
		"status", p.Status,
	)
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	setupLogger(cfg.Log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var db storage.Writer
	db, err = storage.NewInfluxDB(cfg.Storage)
	if err != nil {
		slog.Error("failed to connect to InfluxDB", "error", err)
		os.Exit(1)
	}

	payloadChan := make(chan telemetry.Payload, telemetryQueueSize)

	mqttBroker, err := broker.NewMQTTBroker(cfg.Broker, payloadChan)
	if err != nil {
		_ = db.Close()
		slog.Error("failed to connect to MQTT", "error", err)
		os.Exit(1)
	}

	var wg sync.WaitGroup
	wg.Go(func() {
		processTelemetry(db, payloadChan)
	})

	if err := mqttBroker.StartSubscription(cfg.Broker.Topic); err != nil {
		mqttBroker.Disconnect()
		close(payloadChan)
		wg.Wait()
		_ = db.Close()
		slog.Error("failed to subscribe to MQTT topic", "topic", cfg.Broker.Topic, "error", err)
		os.Exit(1)
	}
	slog.Info("worker listening to MQTT topic", "topic", cfg.Broker.Topic, "client_id", cfg.Broker.ClientID)

	<-ctx.Done()
	slog.Info("shutdown signal received, starting graceful shutdown...")

	// 1. Desconectar o broker MQTT (para recebimento de novas mensagens)
	mqttBroker.Disconnect()
	slog.Info("MQTT disconnected")

	// 2. Fechar o canal para sinalizar à goroutine para drenar as mensagens restantes
	close(payloadChan)

	// 3. Aguardar todas as gravações pendentes no InfluxDB finalizarem
	wg.Wait()

	// 4. Fechar conexão com InfluxDB com segurança
	if err := db.Close(); err != nil {
		slog.Error("failed to close InfluxDB connection", "error", err)
	} else {
		slog.Info("InfluxDB connection closed successfully")
	}

	slog.Info("worker shut down safely")
}
