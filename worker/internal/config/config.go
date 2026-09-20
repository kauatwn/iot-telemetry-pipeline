package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	Storage StorageConfig
	Broker  BrokerConfig
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

// loadDotEnv lê um arquivo .env e injeta as variáveis no ambiente do processo caso ainda não existam.
func loadDotEnv(filepath string) {
	f, err := os.Open(filepath)
	if err != nil {
		return // Arquivo .env é opcional em ambientes de produção/container
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		val = strings.Trim(val, `"'`)
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, val)
		}
	}
}

func Load() (*Config, error) {
	loadDotEnv(".env")

	clientID := os.Getenv("MQTT_CLIENT_ID")
	if clientID == "" {
		clientID = fmt.Sprintf("go_worker_%d", time.Now().UnixNano()%1000000)
	}

	topic := os.Getenv("MQTT_TOPIC")
	if topic == "" {
		topic = "telemetry/temperature"
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
