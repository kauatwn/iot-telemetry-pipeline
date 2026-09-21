package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Storage StorageConfig
	Broker  BrokerConfig
	Log     LogConfig
}

type StorageConfig struct {
	URL   string
	Token string
	DB    string
}

type BrokerConfig struct {
	URL      string
	User     string
	Password string
	ClientID string
	Topic    string
}

type LogConfig struct {
	Level  string
	Format string
}

func Load() (*Config, error) {
	// Carrega o arquivo .env (opcional em ambientes de produção/contêineres)
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		slog.Warn("failed to load .env file", "error", err)
	}

	clientID := os.Getenv("MQTT_CLIENT_ID")
	if clientID == "" {
		clientID = fmt.Sprintf("go_worker_%d", time.Now().UnixNano()%1000000)
	}

	topic := os.Getenv("MQTT_TOPIC")
	if topic == "" {
		topic = "telemetry/temperature"
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "INFO"
	}

	logFormat := os.Getenv("LOG_FORMAT")
	if logFormat == "" {
		logFormat = "text"
	}

	cfg := &Config{
		Storage: StorageConfig{
			URL:   os.Getenv("INFLUX_URL"),
			Token: os.Getenv("INFLUX_TOKEN"),
			DB:    os.Getenv("INFLUX_DB"),
		},
		Broker: BrokerConfig{
			URL:      os.Getenv("MQTT_URL"),
			User:     os.Getenv("MQTT_USER"),
			Password: os.Getenv("MQTT_PASSWORD"),
			ClientID: clientID,
			Topic:    topic,
		},
		Log: LogConfig{
			Level:  logLevel,
			Format: logFormat,
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	var errs []error

	if c.Storage.URL == "" {
		errs = append(errs, errors.New("influx url is required (INFLUX_URL)"))
	}
	if c.Storage.Token == "" {
		errs = append(errs, errors.New("influx token is required (INFLUX_TOKEN)"))
	}
	if c.Storage.DB == "" {
		errs = append(errs, errors.New("influx db is required (INFLUX_DB)"))
	}

	if c.Broker.URL == "" {
		errs = append(errs, errors.New("mqtt url is required (MQTT_URL)"))
	}
	if c.Broker.User == "" {
		errs = append(errs, errors.New("mqtt user is required (MQTT_USER)"))
	}
	if c.Broker.Password == "" {
		errs = append(errs, errors.New("mqtt password is required (MQTT_PASSWORD)"))
	}

	return errors.Join(errs...)
}
