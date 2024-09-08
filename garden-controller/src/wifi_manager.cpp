#include "wifi_manager.h"
#include "config.h"
#include "wifi_config.h"

char* mqtt_server = new char();
char* mqtt_topic_prefix = new char();
int mqtt_port;

WiFiManagerParameter custom_mqtt_server("server", "mqtt server", "192.168.0.x", 40);
WiFiManagerParameter custom_mqtt_topic_prefix("topic_prefix", "mqtt topic prefix", "garden", 40);
WiFiManagerParameter custom_mqtt_port("port", "mqtt port", "1883", 6);

WiFiManager wifiManager;

TaskHandle_t wifiManagerLoopTaskHandle;

void saveParamsToConfig() {
  // read updated parameters
  strcpy(mqtt_server, custom_mqtt_server.getValue());
  strcpy(mqtt_topic_prefix, custom_mqtt_topic_prefix.getValue());
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
  printf("setting up filesystem\n");

  if (!LittleFS.begin(FORMAT_LITTLEFS_IF_FAILED)) {
    printf("failed to mount FS\n");
    return;
  }
  printf("successfully mounted FS\n");

  if (!LittleFS.exists("/config.json")) {
    printf("config doesn't exist\n");
    return;
  }
  printf("config file exists\n");

  // file exists, reading and loading
  File configFile = LittleFS.open("/config.json", "r");
  if (!configFile) {
    return;
  }
  printf("opened config file\n");

  size_t size = configFile.size();

  // Allocate a buffer to store contents of the file.
  std::unique_ptr<char[]> buf(new char[size]);

  configFile.readBytes(buf.get(), size);
  configFile.close();

  DynamicJsonDocument json(1024);
  auto deserializeError = deserializeJson(json, buf.get());
  if (deserializeError) {
    printf("failed to load json config\n");
    return;
  }
  strcpy(mqtt_server, json["mqtt_server"]);
  strcpy(mqtt_topic_prefix, json["mqtt_topic_prefix"]);
  mqtt_port = json["mqtt_port"];

  printf("loaded config JSON: %s %s %d\n", mqtt_server, mqtt_topic_prefix, mqtt_port);
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

void wifiDisconnectHandler(WiFiEvent_t event, WiFiEventInfo_t info) {
    ESP.restart();
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

  char hostname[50];
  snprintf(hostname, sizeof(hostname), "%s-controller", mqtt_topic_prefix);
  wifiManager.setHostname(hostname);

  // wifiManager.resetSettings();

  setupFS();

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

  // Create event handler tp reconnect to WiFi
  WiFi.onEvent(wifiDisconnectHandler, WiFiEvent_t::ARDUINO_EVENT_WIFI_STA_DISCONNECTED);
}
