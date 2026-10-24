package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"worker/internal/broker"
	"worker/internal/config"
	"worker/internal/storage"
	"worker/internal/telemetry"
)

const telemetryQueueSize = 100

func main() {
	// 1. Logger padronizado em JSON no stdout
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// 2. Captura idiomática de sinais do SO
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 3. Instanciação de adaptadores e domínio (Wiring)
	db, err := storage.NewInfluxDB(cfg.Storage)
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

	processor := telemetry.NewProcessor(db)

	// 4. Execução assíncrona não-bloqueante
	var wg sync.WaitGroup
	wg.Go(func() {
		processor.Process(payloadChan)
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

	// 5. Bloqueio até receber sinal de encerramento
	<-ctx.Done()
	slog.Info("shutdown signal received, starting graceful shutdown...")

	// 6. Encerramento gracioso em ordem estrita
	mqttBroker.Disconnect()
	slog.Info("MQTT disconnected")

	close(payloadChan)
	wg.Wait()

	if err := db.Close(); err != nil {
		slog.Error("failed to close InfluxDB connection", "error", err)
	} else {
		slog.Info("InfluxDB connection closed successfully")
	}

	slog.Info("worker shut down safely")
}
