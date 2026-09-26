#pragma once

#include <cstdint>

// Copie este arquivo para 'include/secrets.h' e preencha com suas credenciais reais.
// O arquivo 'secrets.h' é ignorado pelo Git para evitar vazamento de segredos.

// Configurações de rede Wi-Fi (No simulador Wokwi, utilizar SSID "Wokwi-GUEST" e senha vazia)
constexpr char default_wifi_ssid[] = "Wokwi-GUEST";
constexpr char default_wifi_password[] = "";

// Configurações do broker HiveMQ Cloud (porta 8883 / TLS)
constexpr char default_mqtt_broker_host[] = "seu_cluster_id.s1.eu.hivemq.cloud";
constexpr uint16_t default_mqtt_broker_port = 8883;
constexpr char default_mqtt_username[] = "SEU_USUARIO_HIVEMQ";
constexpr char default_mqtt_password[] = "SUA_SENHA_HIVEMQ";
