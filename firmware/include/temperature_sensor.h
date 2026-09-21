#pragma once

#include "telemetry.h"

void sensor_init();

float sensor_read_celsius(SensorReadStatus& status);
