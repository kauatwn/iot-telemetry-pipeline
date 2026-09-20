/*
 * Sistema de Coleta Térmica e Telemetria IoT com ESP32
 *
 * Descrição do Projeto:
 * Firmware para aquisição contínua de temperatura ambiente e transmissão de dados para a nuvem.
 * O sistema realiza a amostragem digital do sensor DS18B20 via barramento 1-Wire, valida a integridade física da
 * leitura contra falhas de conexão, formata as informações no contrato JSON padronizado e transmite os pacotes
 * periodicamente a um broker MQTT (HiveMQ Cloud) sobre canal criptografado TLS.
 *
 * Regras de Negócio e Comportamento Operacional:
 * 1. Aquisição e Validação Térmica (Sensor DS18B20 no GPIO 14):
 *    - Amostragem digital através do barramento Dallas 1-Wire utilizando resistor de pull-up externo de 4.7 kΩ ao 3V3.
 *    - Detecção de falhas de hardware: leituras iguais a -127.0 °C (DEVICE_DISCONNECTED_C) identificam desconexão ou
 * circuito aberto.
 *    - Validação de limites físicos nominais (-55.0 °C a +125.0 °C). Valores fora do intervalo são marcados como erro
 * de leitura.
 * 2. Comunicação Segura e Resiliência de Nuvem (HiveMQ Cloud via MQTT/TLS):
 *    - Conexão Wi-Fi com controle de timeout na inicialização para evitar bloqueio indefinido do sistema.
 *    - Criptografia ponta a ponta na camada de transporte (porta 8883) com autenticação por usuário e senha.
 *    - Cliente seguro configurado em modo setInsecure() para validação simplificada em ambiente de
 * simulação/desenvolvimento.
 *    - Reconexão resiliente: em caso de queda de link ou instabilidade, tentativas periódicas a cada 5 segundos ocorrem
 *      sem interromper a rotina operacional principal.
 * 3. Formatação e Transporte de Dados (Contrato de Telemetria JSON):
 *    - Amostragem e transmissão periódica a cada 2000 ms (2.0 segundos), controlada via temporização não bloqueante com
 * millis().
 *    - Payload serializado em JSON com campos padronizados para ingestão em banco de séries temporais:
 *      device_id, sensor, temperature, unit, uptime_ms e status.
 *    - Publicação no tópico MQTT 'telemetry/temperature'.
 *    - Diagnóstico de operação e carga útil espelhados no Monitor Serial a 115200 bps para depuração local.
 */

#include <Arduino.h>
#include <ArduinoJson.h>
#include <DallasTemperature.h>
#include <OneWire.h>
#include <PubSubClient.h>
#include <WiFi.h>
#include <WiFiClientSecure.h>

#include "secrets.h"

namespace {
// Mapeamento de pinos do hardware
constexpr uint8_t pin_one_wire_bus = 14;  // Entrada digital: sensor de temperatura DS18B20 (barramento 1-Wire)

// Parâmetros do sensor de temperatura DS18B20 (°C)
constexpr float ds18b20_min_temp_c = -55.0F;                       // Limite mínimo de temperatura
constexpr float ds18b20_max_temp_c = 125.0F;                       // Limite máximo de temperatura
constexpr float ds18b20_disconnect_value = DEVICE_DISCONNECTED_C;  // Leitura com sensor desconectado (-127.0 °C)

// Configurações de rede Wi-Fi e broker MQTT (carregadas de include/secrets.h)
constexpr auto wifi_ssid = default_wifi_ssid;
constexpr auto wifi_password = default_wifi_password;
constexpr auto mqtt_broker_host = default_mqtt_broker_host;
constexpr uint16_t mqtt_broker_port = default_mqtt_broker_port;
constexpr auto mqtt_username = default_mqtt_username;
constexpr auto mqtt_password = default_mqtt_password;
constexpr auto mqtt_client_id = "esp32_sensor_device_01";
constexpr auto mqtt_topic_telemetry = "telemetry/temperature";

// Identificação do dispositivo e telemetria
constexpr auto telemetry_device_id = "esp32-sensor-01";
constexpr auto telemetry_sensor_model = "ds18b20";
constexpr auto telemetry_unit = "celsius";

// Temporizações e comunicação serial
constexpr unsigned long serial_baud_rate = 115200;        // Velocidade da porta serial (115200 bps)
constexpr unsigned long telemetry_interval_ms = 2000;     // Intervalo de transmissão da telemetria (2s)
constexpr unsigned long mqtt_reconnect_retry_ms = 5000;   // Intervalo entre tentativas de reconexão MQTT (5s)
constexpr unsigned long wifi_reconnect_retry_ms = 10000;  // Intervalo entre tentativas de reconexão Wi-Fi (10s)
constexpr unsigned long wifi_connect_timeout_ms = 10000;  // Timeout de conexão Wi-Fi inicial (10s)
constexpr uint8_t telemetry_temp_decimals = 2;            // Casas decimais da temperatura na serial
constexpr size_t json_payload_buffer_size = 256;          // Buffer fixo na pilha (evita fragmentação de Heap)

// Estados operacionais da leitura de temperatura
enum class SensorReadStatus : uint8_t {
  Success,       // Leitura realizada com sucesso
  Disconnected,  // Sensor desconectado (-127 °C)
  OutOfBounds,   // Leitura fora dos limites físicos (-55 °C a 125 °C)
};

// Estrutura de dados para agregação e transporte da telemetria
struct TemperatureTelemetry {
  float temperature_c;
  SensorReadStatus status;
  unsigned long timestamp_ms;
};

// Instâncias de periféricos e comunicação de rede
OneWire one_wire_bus(pin_one_wire_bus);
DallasTemperature dallas_sensor(&one_wire_bus);
WiFiClientSecure secure_wifi_client;
PubSubClient mqtt_client(secure_wifi_client);

// Variáveis de estado global do sistema
unsigned long last_telemetry_ms = 0;
unsigned long last_mqtt_reconnect_attempt_ms = 0;
unsigned long last_wifi_reconnect_attempt_ms = 0;

// Retorna o rótulo textual do status do sensor
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

// Inicialização e conexão inicial com a rede Wi-Fi
void setup_wifi() {
  Serial.println();
  Serial.print(F("[WIFI] Conectando a rede: "));
  Serial.println(wifi_ssid);

  WiFiClass::mode(WIFI_STA);
  WiFi.begin(wifi_ssid, wifi_password);

  const unsigned long start_attempt_time = millis();
  while (WiFiClass::status() != WL_CONNECTED && millis() - start_attempt_time < wifi_connect_timeout_ms) {
    delay(250);
    Serial.print('.');
  }

  if (WiFiClass::status() == WL_CONNECTED) {
    Serial.println();
    Serial.print(F("[WIFI] Conectado com sucesso! Endereço IP: "));
    Serial.println(WiFi.localIP());
  } else {
    Serial.println();
    Serial.println(F("[WIFI] Falha na conexão inicial (timeout). O sistema continuará tentando no loop."));
  }
}

// Reconexão periódica resiliente de Wi-Fi sem bloquear a execução principal
void maintain_wifi_connection(const unsigned long current_ms) {
  if (WiFiClass::status() == WL_CONNECTED) {
    return;
  }

  if (current_ms - last_wifi_reconnect_attempt_ms >= wifi_reconnect_retry_ms) {
    last_wifi_reconnect_attempt_ms = current_ms;
    Serial.println(F("[WIFI] Conexão perdida. Tentando reconectar ao Wi-Fi..."));
    WiFi.disconnect();
    WiFi.reconnect();
  }
}

// Conexão e autenticação com o broker HiveMQ Cloud via TLS
bool connect_to_mqtt() {
  Serial.print(F("[MQTT] Conectando ao HiveMQ Cloud ("));
  Serial.print(mqtt_broker_host);
  Serial.print(':');
  Serial.print(mqtt_broker_port);
  Serial.println(F(")..."));

  if (mqtt_client.connect(mqtt_client_id, mqtt_username, mqtt_password)) {
    Serial.println(F("[MQTT] Conectado com sucesso ao broker HiveMQ Cloud!"));
    return true;
  }

  Serial.print(F("[MQTT] Falha na conexão. Código de erro MQTT rc="));
  Serial.println(mqtt_client.state());
  return false;
}

// Gerencia a reconexão automática ao broker MQTT de forma não bloqueante
void maintain_mqtt_connection(const unsigned long current_ms) {
  if (WiFiClass::status() != WL_CONNECTED) {
    return;
  }

  if (mqtt_client.connected()) {
    return;
  }

  if (current_ms - last_mqtt_reconnect_attempt_ms >= mqtt_reconnect_retry_ms) {
    last_mqtt_reconnect_attempt_ms = current_ms;
    connect_to_mqtt();
  }
}

// Leitura da temperatura com amostragem assíncrona não-bloqueante
float read_temperature_celsius(SensorReadStatus& status) {
  const float temp_c = dallas_sensor.getTempCByIndex(0);
  dallas_sensor.requestTemperatures();  // Dispara assincronamente a próxima conversão

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

// Serializa os dados de telemetria em buffer estático (zero alocação dinâmica / sem fragmentação de Heap)
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

// Publica a mensagem de telemetria no tópico MQTT
bool publish_telemetry(const char* payload) {
  if (!mqtt_client.connected()) {
    return false;
  }
  return mqtt_client.publish(mqtt_topic_telemetry, payload);
}

// Exibe os dados de telemetria no Monitor Serial
void print_telemetry_serial(const TemperatureTelemetry& telemetry, const char* payload, const bool published) {
  Serial.print(F("[TELEMETRIA] "));
  if (telemetry.status == SensorReadStatus::Success) {
    Serial.print(F("Temp: "));
    Serial.print(telemetry.temperature_c, telemetry_temp_decimals);
    Serial.print(F(" °C"));
  } else {
    Serial.print(F("Status Sensor: "));
    Serial.print(get_sensor_status_name(telemetry.status));
  }

  Serial.print(F(" | MQTT: "));
  Serial.print(published ? F("ENVIADO COM SUCESSO") : F("FALHA/DESCONECTADO"));
  Serial.print(F(" | Payload: "));
  Serial.println(payload);
}
}  // namespace

void setup() {
  Serial.begin(serial_baud_rate);
  delay(500);

  Serial.println(F("=================================================="));
  Serial.println(F(" PIPELINE DE TELEMETRIA IOT - FIRMWARE ESP32     "));
  Serial.println(F(" Protocolo: OneWire (DS18B20) + MQTT/TLS (HiveMQ)"));
  Serial.println(F(" Status: Inicializado com Sucesso                "));
  Serial.println(F("=================================================="));

  // Inicialização do barramento 1-Wire e sensor DS18B20 em modo não-bloqueante
  dallas_sensor.begin();
  dallas_sensor.setWaitForConversion(false);  // Desativa espera de 750ms bloqueante
  dallas_sensor.requestTemperatures();        // Dispara a primeira amostragem
  Serial.print(F("[HARDWARE] Dispositivos 1-Wire encontrados: "));
  Serial.println(dallas_sensor.getDeviceCount());

  // Configuração TLS para o HiveMQ Cloud (modo sem verificação de certificado raiz)
  secure_wifi_client.setInsecure();

  // Configuração do broker MQTT
  mqtt_client.setServer(mqtt_broker_host, mqtt_broker_port);

  // Conexão à rede Wi-Fi
  setup_wifi();

  Serial.println(F("[SISTEMA] Inicialização concluída. Entrando no loop operacional."));
  Serial.println(F("=================================================="));
}

void loop() {
  const unsigned long current_ms = millis();

  // Gerencia a resiliência das conexões de rede de forma não-bloqueante
  maintain_wifi_connection(current_ms);
  maintain_mqtt_connection(current_ms);

  // Processa a fila de mensagens MQTT
  if (mqtt_client.connected()) {
    mqtt_client.loop();
  }

  // Transmissão periódica da telemetria a cada 2 segundos
  if (current_ms - last_telemetry_ms >= telemetry_interval_ms) {
    last_telemetry_ms = current_ms;

    // Leitura da temperatura via sensor DS18B20
    auto read_status = SensorReadStatus::Success;
    const float temperature_c = read_temperature_celsius(read_status);

    // Agregação dos dados no pacote de telemetria
    const TemperatureTelemetry telemetry = {
        .temperature_c = temperature_c,
        .status = read_status,
        .timestamp_ms = current_ms,
    };

    // Serialização do pacote em buffer estático na pilha
    char json_payload[json_payload_buffer_size];
    if (serialize_telemetry_json(telemetry, json_payload, sizeof(json_payload))) {
      // Publicação no broker MQTT
      const bool published = publish_telemetry(json_payload);

      // Exibição dos dados no Monitor Serial
      print_telemetry_serial(telemetry, json_payload, published);
    } else {
      Serial.println(F("[ERRO] Falha na serialização do JSON: buffer estático insuficiente."));
    }
  }
}
