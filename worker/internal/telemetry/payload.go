package telemetry

type Payload struct {
	DeviceID    string   `json:"device_id"`
	Sensor      string   `json:"sensor"`
	Temperature *float64 `json:"temperature"`
	Unit        string   `json:"unit"`
	UptimeMs    int64    `json:"uptime_ms"`
	Status      string   `json:"status"`
}

// IsDisconnected verifica se a leitura indica que o sensor de hardware está desconectado.
func (p Payload) IsDisconnected() bool {
	return p.Temperature == nil || p.Status == "DISCONNECTED"
}
