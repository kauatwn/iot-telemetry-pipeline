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

func setupLogger() {
	var level slog.Level
	switch strings.ToUpper(os.Getenv("LOG_LEVEL")) {
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
	if strings.ToLower(os.Getenv("LOG_FORMAT")) == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(handler))
}

func main() {
	setupLogger()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("falha ao carregar configuração", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var db storage.Writer
	db, err = storage.NewInfluxDB(cfg.Storage)
	if err != nil {
		slog.Error("erro ao conectar no InfluxDB", "error", err)
		os.Exit(1)
	}

	payloadChan := make(chan telemetry.Payload, 100)

	mqttBroker, err := broker.NewMQTTBroker(cfg.Broker, payloadChan)
	if err != nil {
		_ = db.Close()
		slog.Error("erro ao conectar no MQTT", "error", err)
		os.Exit(1)
	}

	var wg sync.WaitGroup
	wg.Go(func() {
		slog.Info("aguardando mensagens de telemetria na esteira...")
		for payload := range payloadChan {
			if payload.HasTemperature() {
				slog.Info("processando telemetria",
					"device_id", payload.DeviceID,
					"sensor", payload.Sensor,
					"temperatura", *payload.Temperature,
					"unit", payload.Unit,
					"status", payload.Status,
				)
			} else {
				slog.Warn("processando telemetria (sensor desconectado)",
					"device_id", payload.DeviceID,
					"sensor", payload.Sensor,
					"unit", payload.Unit,
					"status", payload.Status,
				)
			}

			writeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := db.Save(writeCtx, payload); err != nil {
				slog.Error("falha ao salvar telemetria no InfluxDB",
					"error", err,
					"device_id", payload.DeviceID,
				)
			}
			cancel()
		}
		slog.Info("goroutine consumidora finalizou a drenagem do canal")
	})

	if err := mqttBroker.StartSubscription(cfg.Broker.Topic); err != nil {
		mqttBroker.Disconnect()
		close(payloadChan)
		wg.Wait()
		_ = db.Close()
		slog.Error("erro ao assinar o tópico MQTT", "topic", cfg.Broker.Topic, "error", err)
		os.Exit(1)
	}
	slog.Info("worker escutando tópico MQTT", "topic", cfg.Broker.Topic, "client_id", cfg.Broker.ClientID)

	<-ctx.Done()
	slog.Info("sinal de encerramento recebido, iniciando shutdown gracioso...")

	// 1. Desconectar o broker MQTT (para recebimento de novas mensagens)
	mqttBroker.Disconnect()
	slog.Info("MQTT desconectado")

	// 2. Fechar o canal para sinalizar à goroutine para drenar as mensagens restantes
	close(payloadChan)

	// 3. Aguardar todas as gravações pendentes no InfluxDB finalizarem
	wg.Wait()

	// 4. Fechar conexão com InfluxDB com segurança
	if err := db.Close(); err != nil {
		slog.Error("erro ao fechar conexão com InfluxDB", "error", err)
	} else {
		slog.Info("conexão com InfluxDB encerrada com sucesso")
	}

	slog.Info("worker encerrado com segurança")
}
