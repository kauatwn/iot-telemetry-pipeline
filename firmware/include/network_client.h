#pragma once

void network_init();

void network_maintain(unsigned long current_ms);

void network_loop();

bool network_publish(const char* payload);
