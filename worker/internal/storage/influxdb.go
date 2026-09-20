package storage

import (
	"context"
	"fmt"
	"time"
	"worker/internal/config"
	"worker/internal/telemetry"

	"github.com/InfluxCommunity/influxdb3-go/v2/influxdb3"
)

// Writer define a interface de persistência de telemetria
type Writer interface {
	Save(ctx context.Context, p telemetry.Payload) error
	Close() error
}

type InfluxDB struct {
	client *influxdb3.Client
}

var _ Writer = (*InfluxDB)(nil)

func NewInfluxDB(cfg config.StorageConfig) (*InfluxDB, error) {
	client, err := influxdb3.New(influxdb3.ClientConfig{
		Host:     cfg.URL,
		Token:    cfg.Token,
		Database: cfg.DB,
	})
	if err != nil {
		return nil, fmt.Errorf("create influxdb3 client: %w", err)
	}

	return &InfluxDB{client: client}, nil
}

func (db *InfluxDB) Save(ctx context.Context, p telemetry.Payload) error {
	point := influxdb3.NewPointWithMeasurement("sensor_ambiente").
		SetTag("device_id", p.DeviceID).
		SetTag("sensor_model", p.Sensor).
		SetTag("status", p.Status).
		SetStringField("status", p.Status).
		SetTimestamp(time.Now())

	if p.Unit != "" {
		point.SetTag("unit", p.Unit)
	}

	if p.UptimeMs > 0 {
		point.SetIntegerField("uptime_ms", p.UptimeMs)
	}

	if p.HasTemperature() {
		point.SetDoubleField("temperatura", *p.Temperature)
	}

	if err := db.client.WritePoints(ctx, []*influxdb3.Point{point}); err != nil {
		return fmt.Errorf("write telemetry point: %w", err)
	}

	return nil
}

func (db *InfluxDB) Close() error {
	if err := db.client.Close(); err != nil {
		return fmt.Errorf("close influxdb client: %w", err)
	}
	return nil
}
