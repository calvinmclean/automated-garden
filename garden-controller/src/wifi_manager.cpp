#include "wifi_manager.h"
#include "config.h"

#if __has_include("wifi_config.h")
#include "wifi_config.h"
#endif

char mqtt_server[41] = {0};
char mqtt_topic_prefix[41] = {0};
int mqtt_port;

WiFiManagerParameter custom_mqtt_server("server", "mqtt server", "192.168.0.x", 40);
WiFiManagerParameter custom_mqtt_topic_prefix("topic_prefix", "mqtt topic prefix", "garden", 40);
WiFiManagerParameter custom_mqtt_port("port", "mqtt port", "1883", 6);

WiFiManager wifiManager;

TaskHandle_t wifiManagerLoopTaskHandle;

const unsigned long WIFI_RESTART_TIMEOUT_MS = 5UL * 60UL * 1000UL;

void saveParamsToConfig() {
  // read updated parameters
  strlcpy(mqtt_server, custom_mqtt_server.getValue(), sizeof(mqtt_server));
  strlcpy(mqtt_topic_prefix, custom_mqtt_topic_prefix.getValue(), sizeof(mqtt_topic_prefix));
  mqtt_port = atoi(custom_mqtt_port.getValue());

  DynamicJsonDocument json(1024);
  json["mqtt_server"] = mqtt_server;
  json["mqtt_port"] = mqtt_port;
  json["mqtt_topic_prefix"] = mqtt_topic_prefix;

  File configFile = LittleFS.open("/config.json", "w");
  if (!configFile) {
    printf("failed to open config file for writing\n");
  }

  serializeJson(json, configFile);
  configFile.close();
}

void setupFS() {
  if (!LittleFS.exists("/config.json")) {
    printf("mqtt config doesn't exist\n");
    return;
  }
  printf("mqtt config file exists\n");

  // file exists, reading and loading
  File configFile = LittleFS.open("/config.json", "r");
  if (!configFile) {
    return;
  }
  printf("opened mqtt config file\n");

  size_t size = configFile.size();

  // Allocate a buffer to store contents of the file.
  std::unique_ptr<char[]> buf(new char[size]);

  configFile.readBytes(buf.get(), size);
  configFile.close();

  DynamicJsonDocument json(1024);
  auto deserializeError = deserializeJson(json, buf.get());
  if (deserializeError) {
    printf("failed to load mqtt json config\n");
    return;
  }
  strlcpy(mqtt_server, json["mqtt_server"], sizeof(mqtt_server));
  strlcpy(mqtt_topic_prefix, json["mqtt_topic_prefix"], sizeof(mqtt_topic_prefix));
  mqtt_port = json["mqtt_port"];

  printf("loaded mqtt config: %s %s %d\n", mqtt_server, mqtt_topic_prefix, mqtt_port);
}

/*
  wifiManagerLoopTask runs the WifiManager process loop and restarts the
  controller if it remains disconnected long enough for the WiFi stack to be
  considered wedged. Normal reconnect attempts are handled by the ESP32 WiFi
  stack's auto-reconnect behavior.
*/
void wifiManagerLoopTask(void* parameters) {
    bool disconnected = WiFi.status() != WL_CONNECTED;
    unsigned long disconnectedAt = millis();

    if (disconnected) {
        printf("WiFi watchdog started, restart in %lu ms if connection is not restored\n",
               WIFI_RESTART_TIMEOUT_MS);
    }

    while (true) {
        wifiManager.process();

        bool connected = WiFi.status() == WL_CONNECTED;
        if (connected) {
            if (disconnected) {
                printf("WiFi watchdog cleared after IP connectivity was restored\n");
                disconnected = false;
            }
        } else if (!disconnected) {
            disconnected = true;
            disconnectedAt = millis();
            printf("WiFi watchdog started, restart in %lu ms if connection is not restored\n",
                   WIFI_RESTART_TIMEOUT_MS);
        } else if (millis() - disconnectedAt >= WIFI_RESTART_TIMEOUT_MS) {
            printf("WiFi unavailable for %lu ms, restarting controller...\n",
                   WIFI_RESTART_TIMEOUT_MS);
            ESP.restart();
            vTaskDelay(portMAX_DELAY);
        }

        vTaskDelay(5 / portTICK_PERIOD_MS);
    }
    vTaskDelete(NULL);
}

void wifiGotIPHandler(WiFiEvent_t event, WiFiEventInfo_t info) {
    printf("WiFi IP connectivity restored, ip=%s\n", WiFi.localIP().toString().c_str());

    MDNS.end();
    if (!MDNS.begin(mqtt_topic_prefix)) {
        printf("error restarting mDNS after reconnect\n");
    } else {
        MDNS.addService("http", "tcp", 80);
    }
}

void wifiDisconnectHandler(WiFiEvent_t event, WiFiEventInfo_t info) {
    printf("WiFi disconnected, reason=%u; ESP32 auto-reconnect remains active\n",
           info.wifi_sta_disconnected.reason);
}

#if defined SSID && defined PASSWORD
/* connect directly to WiFi and run web portal in the background */
void connectWifiDirect() {
    printf("Connecting to %s as %s\n", SSID, mqtt_topic_prefix);
    WiFi.begin(SSID, PASSWORD);

    while (WiFi.status() != WL_CONNECTED) {
        delay(500);
        printf(".");
    }

    printf("Wifi connected...\n");

    strlcpy(mqtt_server, MQTT_ADDRESS, sizeof(mqtt_server));
    strlcpy(mqtt_topic_prefix, TOPIC_PREFIX, sizeof(mqtt_topic_prefix));
    mqtt_port = MQTT_PORT;

    wifiManager.setEnableConfigPortal(false);
    wifiManager.setConfigPortalBlocking(false);
    wifiManager.autoConnect();
}
#endif

void runWifiManagerPortal() {
    bool connected = wifiManager.autoConnect("GardenControllerSetup", "password");
    if (!connected) {
      printf("failed to connect and hit timeout\n");
      delay(3000);
      ESP.restart();
      delay(5000);
    }
}

void setupWifiManager() {
  // Load any previously-saved config first so the setup portal can show
  // the currently-set values instead of the hardcoded defaults.
  setupFS();

  if (mqtt_server[0] != '\0') {
    custom_mqtt_server.setValue(mqtt_server, sizeof(mqtt_server));
  }
  if (mqtt_topic_prefix[0] != '\0') {
    custom_mqtt_topic_prefix.setValue(mqtt_topic_prefix, sizeof(mqtt_topic_prefix));
  }
  if (mqtt_port != 0) {
    char port_str[7];
    snprintf(port_str, sizeof(port_str), "%d", mqtt_port);
    custom_mqtt_port.setValue(port_str, 6);
  }

  wifiManager.setSaveConfigCallback(saveParamsToConfig);
  wifiManager.setSaveParamsCallback(saveParamsToConfig);

  wifiManager.addParameter(&custom_mqtt_server);
  wifiManager.addParameter(&custom_mqtt_topic_prefix);
  wifiManager.addParameter(&custom_mqtt_port);

  // wifiManager.resetSettings();

  wifiManager.setHostname(mqtt_topic_prefix);
  WiFi.setAutoReconnect(true);

  // If SSID/PASSWORD are configured, connect regularly and use WifiManager for setup portal only
  #if defined SSID && defined PASSWORD
  connectWifiDirect();
  #else
  // Otherwise, use WifiManager autoconnect portal
  runWifiManagerPortal();
  #endif

  wifiManager.setParamsPage(true);
  wifiManager.setConfigPortalBlocking(false);
  wifiManager.startWebPortal();

  xTaskCreate(wifiManagerLoopTask, "WifiManagerLoopTask", 4096, NULL, 1, &wifiManagerLoopTaskHandle);

  if (!MDNS.begin(mqtt_topic_prefix)) {
    printf("error starting mDNS\n");
  } else {
    MDNS.addService("http", "tcp", 80);
  }

  WiFi.onEvent(wifiGotIPHandler, WiFiEvent_t::ARDUINO_EVENT_WIFI_STA_GOT_IP);
  WiFi.onEvent(wifiDisconnectHandler, WiFiEvent_t::ARDUINO_EVENT_WIFI_STA_DISCONNECTED);
}
