#include "garden_config.h"
#include <LittleFS.h>

// Write Config to JSON
void serializeConfig(const Config& config, String& jsonString) {
    DynamicJsonDocument doc(2048);

    doc["num_zones"] = config.numZones;
    for (int i = 0; i < config.numZones; i++) {
        doc["valve_pins"][i] = config.valvePins[i];
        doc["pump_pins"][i] = config.pumpPins[i];
    }

    doc["light"] = config.light;
    doc["light_pin"] = config.lightPin;

    doc["fan"] = config.fan;
    doc["fan_pin"] = config.fanPin;

    doc["num_sensors"] = config.numSensors;
    for (int i = 0; i < config.numSensors; i++) {
        doc["sensors"][i]["id"] = config.sensors[i].id;
        doc["sensors"][i]["type"] = config.sensors[i].type;
        doc["sensors"][i]["pin"] = config.sensors[i].pin;
        doc["sensors"][i]["interval"] = config.sensors[i].interval;
    }

    serializeJson(doc, jsonString);
}

// Read Config from JSON
bool deserializeConfig(const char* jsonString, Config& config) {
    DynamicJsonDocument doc(2048);

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

    config.fan = doc["fan"].as<bool>();
    config.fanPin = static_cast<gpio_num_t>(doc["fan_pin"].as<int>());

    config.numSensors = doc["sensors"].size();
    if (config.numSensors > 16) {
        config.numSensors = 16;
    }
    for (int i = 0; i < config.numSensors; i++) {
        strlcpy(config.sensors[i].id, doc["sensors"][i]["id"] | "", sizeof(config.sensors[i].id));
        strlcpy(config.sensors[i].type, doc["sensors"][i]["type"].as<const char*>(), sizeof(config.sensors[i].type));
        config.sensors[i].pin = static_cast<gpio_num_t>(doc["sensors"][i]["pin"].as<int>());
        config.sensors[i].interval = doc["sensors"][i]["interval"].as<int>();
    }

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

    printf("  Fan: %s\n", config.fan ? "Enabled" : "Disabled");
    printf("  Fan Pin: %d\n", (int)config.fanPin);

    printf("  Sensors: %d\n", config.numSensors);
    for (int i = 0; i < config.numSensors; i++) {
        printf("    Sensor %d: type=%s pin=%d interval=%d\n",
               i, config.sensors[i].type, (int)config.sensors[i].pin, config.sensors[i].interval);
    }
}
