#include "temperature_sensor.h"

#include <Arduino.h>
#include <DallasTemperature.h>
#include <OneWire.h>

namespace {
constexpr uint8_t pin_one_wire_bus = 14;
constexpr float ds18b20_min_temp_c = -55.0F;
constexpr float ds18b20_max_temp_c = 125.0F;
constexpr float ds18b20_disconnect_value = DEVICE_DISCONNECTED_C;

OneWire& get_one_wire() {
  static OneWire bus(pin_one_wire_bus);
  return bus;
}

DallasTemperature& get_dallas_sensor() {
  static DallasTemperature sensor(&get_one_wire());
  return sensor;
}
}  // namespace

void sensor_init() {
  auto& dallas_sensor = get_dallas_sensor();
  dallas_sensor.begin();
  dallas_sensor.setWaitForConversion(false);
  dallas_sensor.requestTemperatures();
  Serial.print(F("[HARDWARE] 1-Wire devices found: "));
  Serial.println(dallas_sensor.getDeviceCount());
}

float sensor_read_celsius(SensorReadStatus& status) {
  auto& dallas_sensor = get_dallas_sensor();
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
