#include "garden_config.h"
#include <LittleFS.h>

// Write Config to JSON
void serializeConfig(const Config& config, String& jsonString) {
    DynamicJsonDocument doc(1024);

    doc["num_zones"] = config.numZones;
    for (int i = 0; i < config.numZones; i++) {
        doc["valve_pins"][i] = config.valvePins[i];
        doc["pump_pins"][i] = config.pumpPins[i];
    }

    doc["light"] = config.light;
    doc["light_pin"] = config.lightPin;

    doc["temp_humidity"] = config.tempHumidity;
    doc["temp_humidity_pin"] = config.tempHumidityPin;
    doc["temp_humidity_interval"] = config.tempHumidityInterval;

    serializeJson(doc, jsonString);
}

// Read Config from JSON
bool deserializeConfig(const char* jsonString, Config& config) {
    DynamicJsonDocument doc(1024);

    DeserializationError error = deserializeJson(doc, jsonString);

    if (error) {
        printf("deserialize controller config failed: %s\n", error.c_str());
        return false;
    }

    config.numZones = doc["num_zones"].as<int>();
    for (int i = 0; i < config.numZones; i++) {
        config.valvePins[i] = static_cast<gpio_num_t>(doc["valve_pins"][i].as<int>());
        config.pumpPins[i] = static_cast<gpio_num_t>(doc["pump_pins"][i].as<int>());
    }

    config.light = doc["light"].as<bool>();
    config.lightPin = static_cast<gpio_num_t>(doc["light_pin"].as<int>());

    config.tempHumidity = doc["temp_humidity"].as<bool>();
    config.tempHumidityPin = static_cast<gpio_num_t>(doc["temp_humidity_pin"].as<int>());
    config.tempHumidityInterval = doc["temp_humidity_interval"].as<int>();

    return true;
}

void initFS() {
    printf("setting up filesystem\n");
    if (!LittleFS.begin(true)) {
        printf("failed to mount FS\n");
    }
    printf("successfully mounted FS\n");
}

bool configFileExists() {
    return LittleFS.exists("/garden_config.json");
}

void loadConfigFromFile(Config& config) {
    if (!configFileExists()) {
      printf("controller config doesn't exist\n");
      return;
    }

    File configFile = LittleFS.open("/garden_config.json", "r");
    if (!configFile) {
      return;
    }
    printf("opened controller config file\n");

    size_t size = configFile.size();

    // Allocate a buffer to store contents of the file.
    std::unique_ptr<char[]> buf(new char[size]);

    configFile.readBytes(buf.get(), size);
    configFile.close();

    printf("read controller config file: %s\n", buf.get());

    if (!deserializeConfig(buf.get(), config)) {
      printf("failed to load controller json config\n");
    }
}

void saveConfigToFile(const Config& config) {
  String configJSON;
  serializeConfig(config, configJSON);

  File configFile = LittleFS.open("/garden_config.json", "w");
  if (!configFile) {
    printf("failed to open controller config file for writing\n");
  }

  if (configFile.print(configJSON)) {
    printf("controller config file written successfully\n");
  } else {
    printf("Write failed\n");
  }

  configFile.close();
}

void printConfig(Config& config) {
    printf("Config:\n");
    printf("  Number of Zones: %d\n", config.numZones);

    printf("  Valve/Pump Pins: ");
    for (int i = 0; i < config.numZones; i++) {
        printf("%d/%d ", config.valvePins[i], config.pumpPins[i]);
    }
    printf("\n");

    printf("  Light: %s\n", config.light ? "Enabled" : "Disabled");
    printf("  Light Pin: %d\n", (int)config.lightPin);

    printf("  TempHumidity: %s\n", config.tempHumidity ? "Enabled" : "Disabled");
    printf("  TempHumidity Pin: %d\n", (int)config.tempHumidityPin);
    printf("  TempHumidity Interval: %d\n", config.tempHumidityInterval);
}
