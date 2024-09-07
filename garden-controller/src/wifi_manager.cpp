#include "wifi_manager.h"
#include "config.h"

char* mqtt_server = new char();
char* mqtt_topic_prefix = new char();
int mqtt_port;

WiFiManagerParameter custom_mqtt_server("server", "mqtt server", "", 40);
WiFiManagerParameter custom_mqtt_topic_prefix("topic_prefix", "mqtt topic prefix", "", 40);
WiFiManagerParameter custom_mqtt_port("port", "mqtt port", "8080", 6);

WiFiManager wifiManager;

TaskHandle_t wifiManagerLoopTaskHandle;

void saveConfig() {
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

  // start with defaults
  strcpy(mqtt_server, MQTT_ADDRESS);
  strcpy(mqtt_topic_prefix, TOPIC_PREFIX);
  mqtt_port = MQTT_PORT;

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

void setupWifiManager() {
  wifiManager.setSaveConfigCallback(saveConfig);
  wifiManager.setSaveParamsCallback(saveConfig);

  wifiManager.addParameter(&custom_mqtt_server);
  wifiManager.addParameter(&custom_mqtt_topic_prefix);
  wifiManager.addParameter(&custom_mqtt_port);

  // wifiManager.resetSettings();

  setupFS();

  if (!wifiManager.autoConnect("GardenControllerSetup", "password")) {
    printf("failed to connect and hit timeout\n");
    delay(3000);
    // reset and try again, or maybe put it to deep sleep
    ESP.restart();
    delay(5000);
  }

  wifiManager.setParamsPage(true);
  wifiManager.setConfigPortalBlocking(false);
  wifiManager.startWebPortal();

  xTaskCreate(wifiManagerLoopTask, "WifiManagerLoopTask", 4096, NULL, 1, &wifiManagerLoopTaskHandle);
}
