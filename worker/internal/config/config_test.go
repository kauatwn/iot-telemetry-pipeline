package config

import (
	"os"
	"strings"
	"testing"

	"github.com/joho/godotenv"
)

func TestConfigValidation_MissingAll(t *testing.T) {
	cfg := &Config{}
	err := cfg.validate()
	if err == nil {
		t.Fatal("expected validation error with empty config, got nil")
	}

	errStr := err.Error()
	expectedMissing := []string{
		"INFLUX_URL",
		"INFLUX_TOKEN",
		"INFLUX_DB",
		"MQTT_URL",
		"MQTT_USER",
		"MQTT_PASSWORD",
	}

	for _, expected := range expectedMissing {
		if !strings.Contains(errStr, expected) {
			t.Errorf("expected mention of %q in accumulated error, got: %s", expected, errStr)
		}
	}
}

func TestConfigValidation_Valid(t *testing.T) {
	cfg := &Config{
		Storage: StorageConfig{
			URL:   "http://localhost:8086",
			Token: "my-token",
			DB:    "iot_telemetry",
		},
		Broker: BrokerConfig{
			URL:      "tls://hivemq.cloud:8883",
			User:     "user",
			Password: "pwd",
			ClientID: "test_worker",
			Topic:    "telemetry/temperature",
		},
	}

	if err := cfg.validate(); err != nil {
		t.Fatalf("expected valid configuration without errors, got: %v", err)
	}
}

func TestConfigLoad_Defaults(t *testing.T) {
	t.Setenv("INFLUX_URL", "http://localhost:8086")
	t.Setenv("INFLUX_TOKEN", "token")
	t.Setenv("INFLUX_DB", "db")
	t.Setenv("MQTT_URL", "ssl://broker:8883")
	t.Setenv("MQTT_USER", "user")
	t.Setenv("MQTT_PASSWORD", "pass")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error loading config: %v", err)
	}

	if cfg.Broker.Topic != "telemetry/temperature" {
		t.Errorf("expected default topic 'telemetry/temperature', got: %q", cfg.Broker.Topic)
	}

	if !strings.HasPrefix(cfg.Broker.ClientID, "go_worker_") {
		t.Errorf("expected client ID with prefix 'go_worker_', got: %q", cfg.Broker.ClientID)
	}
}

func TestLoadDotEnv(t *testing.T) {
	tempFile := t.TempDir() + "/.env"
	content := `# Comentário que deve ser ignorado
TEST_ENV_KEY_1=simple_value
TEST_ENV_KEY_2="quoted_value"
TEST_ENV_KEY_3='single_quoted'
`
	if err := os.WriteFile(tempFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp .env file: %v", err)
	}

	if err := godotenv.Load(tempFile); err != nil {
		t.Fatalf("failed to load temp .env file: %v", err)
	}

	if val := os.Getenv("TEST_ENV_KEY_1"); val != "simple_value" {
		t.Errorf("expected 'simple_value', got: %q", val)
	}
	if val := os.Getenv("TEST_ENV_KEY_2"); val != "quoted_value" {
		t.Errorf("expected 'quoted_value', got: %q", val)
	}
	if val := os.Getenv("TEST_ENV_KEY_3"); val != "single_quoted" {
		t.Errorf("expected 'single_quoted', got: %q", val)
	}
}
