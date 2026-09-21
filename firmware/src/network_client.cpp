#include "network_client.h"

#include <Arduino.h>
#include <PubSubClient.h>
#include <WiFi.h>
#include <WiFiClientSecure.h>

#include "secrets.h"

namespace {
constexpr auto wifi_ssid = default_wifi_ssid;
constexpr auto wifi_password = default_wifi_password;
constexpr auto mqtt_broker_host = default_mqtt_broker_host;
constexpr uint16_t mqtt_broker_port = default_mqtt_broker_port;
constexpr auto mqtt_username = default_mqtt_username;
constexpr auto mqtt_password = default_mqtt_password;
constexpr auto mqtt_client_id = "esp32_sensor_device_01";
constexpr auto mqtt_topic_telemetry = "telemetry/temperature";

constexpr unsigned long mqtt_reconnect_retry_ms = 5000;
constexpr unsigned long wifi_reconnect_retry_ms = 10000;
constexpr unsigned long wifi_connect_timeout_ms = 10000;

WiFiClientSecure secure_wifi_client;
PubSubClient mqtt_client(secure_wifi_client);

unsigned long last_mqtt_reconnect_attempt_ms = 0;
unsigned long last_wifi_reconnect_attempt_ms = 0;

void setup_wifi() {
  Serial.println();
  Serial.print(F("[WIFI] Connecting to network: "));
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
    Serial.print(F("[WIFI] Connected successfully! IP Address: "));
    Serial.println(WiFi.localIP());
  } else {
    Serial.println();
    Serial.println(F("[WIFI] Initial connection failed (timeout). System will retry in loop."));
  }
}

void maintain_wifi_connection(const unsigned long current_ms) {
  if (WiFiClass::status() == WL_CONNECTED) {
    return;
  }

  if (current_ms - last_wifi_reconnect_attempt_ms >= wifi_reconnect_retry_ms) {
    last_wifi_reconnect_attempt_ms = current_ms;
    Serial.println(F("[WIFI] Connection lost. Attempting to reconnect to Wi-Fi..."));
    WiFi.disconnect();
    WiFi.reconnect();
  }
}

bool connect_to_mqtt() {
  Serial.print(F("[MQTT] Connecting to HiveMQ Cloud ("));
  Serial.print(mqtt_broker_host);
  Serial.print(':');
  Serial.print(mqtt_broker_port);
  Serial.println(F(")..."));

  if (mqtt_client.connect(mqtt_client_id, mqtt_username, mqtt_password)) {
    Serial.println(F("[MQTT] Connected successfully to HiveMQ Cloud broker!"));
    return true;
  }

  Serial.print(F("[MQTT] Connection failed. MQTT error code rc="));
  Serial.println(mqtt_client.state());
  return false;
}

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
}  // namespace

void network_init() {
  secure_wifi_client.setInsecure();
  mqtt_client.setServer(mqtt_broker_host, mqtt_broker_port);
  setup_wifi();
}

void network_maintain(const unsigned long current_ms) {
  maintain_wifi_connection(current_ms);
  maintain_mqtt_connection(current_ms);
}

void network_loop() {
  if (mqtt_client.connected()) {
    mqtt_client.loop();
  }
}

bool network_publish(const char* payload) {
  if (!mqtt_client.connected()) {
    return false;
  }
  return mqtt_client.publish(mqtt_topic_telemetry, payload);
}
