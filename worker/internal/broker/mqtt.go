package broker

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"worker/internal/config"
	"worker/internal/telemetry"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MQTTBroker struct {
	client mqtt.Client
	out    chan<- telemetry.Payload
	topic  string
	mu     sync.RWMutex
}

func NewMQTTBroker(cfg config.BrokerConfig, outChan chan<- telemetry.Payload) (*MQTTBroker, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(cfg.URL)
	opts.SetClientID(cfg.ClientID)
	opts.SetUsername(cfg.User)
	opts.SetPassword(cfg.Password)

	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetCleanSession(true)
	opts.SetKeepAlive(30 * time.Second)
	opts.SetPingTimeout(10 * time.Second)

	// Configura TLS quando o protocolo for seguro (HiveMQ Cloud, etc.)
	if strings.HasPrefix(cfg.URL, "ssl://") || strings.HasPrefix(cfg.URL, "tls://") || strings.HasPrefix(cfg.URL, "tcps://") {
		opts.SetTLSConfig(&tls.Config{
			MinVersion: tls.VersionTLS12,
		})
	}

	broker := &MQTTBroker{
		out: outChan,
	}

	opts.SetOnConnectHandler(func(c mqtt.Client) {
		broker.mu.RLock()
		currentTopic := broker.topic
		broker.mu.RUnlock()

		if currentTopic != "" {
			slog.Info("conexão MQTT estabelecida", "topic", currentTopic)
			token := c.Subscribe(currentTopic, 0, broker.messageHandler())
			if token.Wait() && token.Error() != nil {
				slog.Error("falha ao assinar tópico na reconexão MQTT", "topic", currentTopic, "error", token.Error())
			}
		} else {
			slog.Info("conexão MQTT estabelecida com sucesso")
		}
	})

	opts.SetConnectionLostHandler(func(c mqtt.Client, err error) {
		slog.Warn("conexão com broker MQTT perdida; tentando reconectar...", "error", err)
	})

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("connect to mqtt broker %s: %w", cfg.URL, token.Error())
	}

	broker.client = client
	return broker, nil
}

func (b *MQTTBroker) StartSubscription(topic string) error {
	b.mu.Lock()
	b.topic = topic
	b.mu.Unlock()

	token := b.client.Subscribe(topic, 0, b.messageHandler())
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("subscribe to topic %s: %w", topic, token.Error())
	}

	return nil
}

func (b *MQTTBroker) messageHandler() mqtt.MessageHandler {
	return func(client mqtt.Client, msg mqtt.Message) {
		var payload telemetry.Payload
		if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
			slog.Error("erro ao decodificar JSON de telemetria", "error", err)
			return
		}

		if !payload.HasTemperature() {
			slog.Warn("telemetria recebida sem temperatura (sensor desconectado)",
				"device_id", payload.DeviceID,
				"status", payload.Status,
			)
		}

		select {
		case b.out <- payload:
		default:
			slog.Warn("buffer de telemetria lotado; mensagem descartada",
				"device_id", payload.DeviceID,
			)
		}
	}
}

func (b *MQTTBroker) Disconnect() {
	b.mu.RLock()
	currentTopic := b.topic
	b.mu.RUnlock()

	if currentTopic != "" && b.client.IsConnected() {
		_ = b.client.Unsubscribe(currentTopic).WaitTimeout(1 * time.Second)
	}

	b.client.Disconnect(250)
}
