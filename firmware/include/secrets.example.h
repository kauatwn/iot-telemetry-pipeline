#pragma once

#include <cstdint>

// Copie este arquivo para 'include/secrets.h' e preencha com suas credenciais reais.
// O arquivo 'secrets.h' é ignorado pelo Git para proteger suas senhas.

// Configurações de rede Wi-Fi (No Wokwi, usar "Wokwi-GUEST" com senha vazia)
constexpr char default_wifi_ssid[] = "Wokwi-GUEST";
constexpr char default_wifi_password[] = "";

// Configurações do broker HiveMQ Cloud (porta 8883 / TLS)
constexpr char default_mqtt_broker_host[] = "SEU_CLUSTER.s1.eu.hivemq.cloud";
constexpr uint16_t default_mqtt_broker_port = 8883;
constexpr char default_mqtt_username[] = "SEU_USUARIO_HIVEMQ";
constexpr char default_mqtt_password[] = "SUA_SENHA_HIVEMQ";
