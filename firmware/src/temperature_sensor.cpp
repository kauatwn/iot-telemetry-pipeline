#include "temperature_sensor.h"

#include <Arduino.h>
#include <DallasTemperature.h>
#include <OneWire.h>

namespace {
constexpr uint8_t pin_one_wire_bus = 14;
constexpr float ds18b20_min_temp_c = -55.0F;
constexpr float ds18b20_max_temp_c = 125.0F;
constexpr float ds18b20_disconnect_value = DEVICE_DISCONNECTED_C;

OneWire one_wire_bus(pin_one_wire_bus);
DallasTemperature dallas_sensor(&one_wire_bus);
}  // namespace

void sensor_init() {
  dallas_sensor.begin();
  dallas_sensor.setWaitForConversion(false);
  dallas_sensor.requestTemperatures();
  Serial.print(F("[HARDWARE] 1-Wire devices found: "));
  Serial.println(dallas_sensor.getDeviceCount());
}

float sensor_read_celsius(SensorReadStatus& status) {
  const float temp_c = dallas_sensor.getTempCByIndex(0);
  dallas_sensor.requestTemperatures();

  if (temp_c == ds18b20_disconnect_value) {
    status = SensorReadStatus::Disconnected;
    return temp_c;
  }

  if (temp_c < ds18b20_min_temp_c || temp_c > ds18b20_max_temp_c) {
    status = SensorReadStatus::OutOfBounds;
    return temp_c;
  }

  status = SensorReadStatus::Success;
  return temp_c;
}
