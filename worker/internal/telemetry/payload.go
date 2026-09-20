package telemetry

type Payload struct {
	DeviceID    string   `json:"device_id"`
	Sensor      string   `json:"sensor"`
	Temperature *float64 `json:"temperature"`
	Unit        string   `json:"unit"`
	UptimeMs    int64    `json:"uptime_ms"`
	Status      string   `json:"status"`
}

// HasTemperature indica se a leitura contém um valor de temperatura válido
func (p Payload) HasTemperature() bool {
	return p.Temperature != nil
}
