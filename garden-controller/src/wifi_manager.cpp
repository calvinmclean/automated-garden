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
  wifiManagerLoopTask will run the WifiManager process loop
*/
void wifiManagerLoopTask(void* parameters) {
    while (true) {
        wifiManager.process();
        vTaskDelay(5 / portTICK_PERIOD_MS);
    }
    vTaskDelete(NULL);
}

static unsigned long lastDisconnectTime = 0;
const unsigned long WIFI_RECONNECT_TIMEOUT_MS = 30000;

void wifiConnectHandler(WiFiEvent_t event, WiFiEventInfo_t info) {
    lastDisconnectTime = 0;
    printf("WiFi connected\n");

    MDNS.end();
    if (!MDNS.begin(mqtt_topic_prefix)) {
        printf("error restarting mDNS after reconnect\n");
    } else {
        MDNS.addService("http", "tcp", 80);
    }
}

void wifiDisconnectHandler(WiFiEvent_t event, WiFiEventInfo_t info) {
    if (lastDisconnectTime == 0) {
        lastDisconnectTime = millis();
    }

    unsigned long disconnectedFor = millis() - lastDisconnectTime;
    printf("WiFi disconnected for %lu ms\n", disconnectedFor);

    if (disconnectedFor > WIFI_RECONNECT_TIMEOUT_MS) {
        printf("WiFi reconnect timeout reached, rebooting...\n");
        ESP.restart();
    }

    WiFi.reconnect();
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
  wifiManager.setSaveConfigCallback(saveParamsToConfig);
  wifiManager.setSaveParamsCallback(saveParamsToConfig);

  wifiManager.addParameter(&custom_mqtt_server);
  wifiManager.addParameter(&custom_mqtt_topic_prefix);
  wifiManager.addParameter(&custom_mqtt_port);

  // wifiManager.resetSettings();

  setupFS();

  wifiManager.setHostname(mqtt_topic_prefix);

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
    return;
  }

  MDNS.addService("http", "tcp", 80);

  WiFi.onEvent(wifiConnectHandler, WiFiEvent_t::ARDUINO_EVENT_WIFI_STA_CONNECTED);
  WiFi.onEvent(wifiDisconnectHandler, WiFiEvent_t::ARDUINO_EVENT_WIFI_STA_DISCONNECTED);
}
