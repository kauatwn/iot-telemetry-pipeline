#pragma once

#include <Arduino.h>

enum class SensorReadStatus : uint8_t {
  Success,
  Disconnected,
  OutOfBounds,
};

struct TemperatureTelemetry {
  float temperature_c;
  SensorReadStatus status;
  unsigned long timestamp_ms;
};

const __FlashStringHelper* get_sensor_status_name(SensorReadStatus status);

bool serialize_telemetry_json(const TemperatureTelemetry& telemetry, char* buffer, size_t max_len);

void print_telemetry_serial(const TemperatureTelemetry& telemetry, const char* payload, bool published);
