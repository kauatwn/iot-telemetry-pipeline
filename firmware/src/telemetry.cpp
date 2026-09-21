#include "telemetry.h"

#include <ArduinoJson.h>

namespace {
constexpr auto telemetry_device_id = "esp32-sensor-01";
constexpr auto telemetry_sensor_model = "ds18b20";
constexpr auto telemetry_unit = "celsius";
constexpr uint8_t telemetry_temp_decimals = 2;
}  // namespace

const __FlashStringHelper* get_sensor_status_name(const SensorReadStatus status) {
  switch (status) {
    case SensorReadStatus::Success:
      return F("OK");
    case SensorReadStatus::Disconnected:
      return F("DISCONNECTED");
    case SensorReadStatus::OutOfBounds:
      return F("OUT_OF_BOUNDS");
  }
  return F("UNKNOWN");
}

bool serialize_telemetry_json(const TemperatureTelemetry& telemetry, char* buffer, const size_t max_len) {
  JsonDocument doc;

  doc["device_id"] = telemetry_device_id;
  doc["sensor"] = telemetry_sensor_model;

  if (telemetry.status == SensorReadStatus::Success) {
    doc["temperature"] = telemetry.temperature_c;
  } else {
    doc["temperature"] = nullptr;
  }

  doc["unit"] = telemetry_unit;
  doc["uptime_ms"] = telemetry.timestamp_ms;
  doc["status"] = get_sensor_status_name(telemetry.status);

  const size_t bytes_written = serializeJson(doc, buffer, max_len);
  return bytes_written > 0 && bytes_written < max_len;
}

void print_telemetry_serial(const TemperatureTelemetry& telemetry, const char* payload, const bool published) {
  Serial.print(F("[TELEMETRY] "));
  if (telemetry.status == SensorReadStatus::Success) {
    Serial.print(F("Temp: "));
    Serial.print(telemetry.temperature_c, telemetry_temp_decimals);
    Serial.print(F(" °C"));
  } else {
    Serial.print(F("Sensor Status: "));
    Serial.print(get_sensor_status_name(telemetry.status));
  }

  Serial.print(F(" | MQTT: "));
  Serial.print(published ? F("SENT SUCCESSFULLY") : F("FAILED/DISCONNECTED"));
  Serial.print(F(" | Payload: "));
  Serial.println(payload);
}
