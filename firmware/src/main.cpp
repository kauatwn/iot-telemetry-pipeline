#include <Arduino.h>

#include "network_client.h"
#include "telemetry.h"
#include "temperature_sensor.h"

namespace {
constexpr unsigned long serial_baud_rate = 115200;
constexpr unsigned long telemetry_interval_ms = 2000;
constexpr size_t json_payload_buffer_size = 256;

unsigned long last_telemetry_ms = 0;
}  // namespace

void setup() {
  Serial.begin(serial_baud_rate);
  delay(500);

  Serial.println(F("=================================================="));
  Serial.println(F(" IOT TELEMETRY PIPELINE - ESP32 FIRMWARE          "));
  Serial.println(F(" Protocol: OneWire (DS18B20) + MQTT/TLS (HiveMQ) "));
  Serial.println(F(" Status: Successfully Initialized                 "));
  Serial.println(F("=================================================="));

  sensor_init();
  network_init();

  Serial.println(F("[SYSTEM] Initialization complete. Entering operational loop."));
  Serial.println(F("=================================================="));
}

void loop() {
  const unsigned long current_ms = millis();

  network_maintain(current_ms);
  network_loop();

  if (current_ms - last_telemetry_ms >= telemetry_interval_ms) {
    last_telemetry_ms = current_ms;

    auto read_status = SensorReadStatus::Success;
    const float temperature_c = sensor_read_celsius(read_status);

    const TemperatureTelemetry telemetry = {
        .temperature_c = temperature_c,
        .status = read_status,
        .timestamp_ms = current_ms,
    };

    char json_payload[json_payload_buffer_size];
    if (serialize_telemetry_json(telemetry, json_payload, sizeof(json_payload))) {
      const bool published = network_publish(json_payload);
      print_telemetry_serial(telemetry, json_payload, published);
    } else {
      Serial.println(F("[ERROR] JSON serialization failed: static buffer insufficient."));
    }
  }
}
