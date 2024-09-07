#include "garden_config.h"

// Write Config to JSON
void serializeConfig(const Config& config, String& jsonString) {
    DynamicJsonDocument doc(1024);

    doc["mqttPort"] = config.mqttPort;
    doc["mqttServer"] = config.mqttServer;
    doc["mqttTopicPrefix"] = config.mqttTopicPrefix;

    doc["numZones"] = config.numZones;
    for (int i = 0; i < config.numZones; i++) {
        doc["zonePins"][i] = config.zonePins[i];
        doc["pumpPins"][i] = config.pumpPins[i];
    }

    doc["light"] = config.light;
    doc["lightPin"] = config.lightPin;

    doc["dht22"] = config.dht22;
    doc["dht22Pin"] = config.dht22Pin;
    doc["dht22Interval"] = config.dht22Interval;

    serializeJson(doc, jsonString);
}

// Read Config from JSON
bool deserializeConfig(const String& jsonString, Config& config) {
    DynamicJsonDocument doc(1024);

    DeserializationError error = deserializeJson(doc, jsonString);

    if (error) {
        printf("deserialize config failed: %s\n", error.c_str());
        return false;
    }

    config.mqttPort = doc["mqttPort"].as<int>();
    config.mqttServer = doc["mqttServer"].as<const char*>();
    config.mqttTopicPrefix = doc["mqttTopicPrefix"].as<const char*>();

    config.numZones = doc["numZones"].as<int>();
    for (int i = 0; i < config.numZones; i++) {
        config.zonePins[i] = static_cast<gpio_num_t>(doc["zonePins"][i].as<int>());
        config.pumpPins[i] = static_cast<gpio_num_t>(doc["pumpPins"][i].as<int>());
    }

    config.light = doc["light"].as<bool>();
    config.lightPin = static_cast<gpio_num_t>(doc["lightPin"].as<int>());

    config.dht22 = doc["dht22"].as<bool>();
    config.dht22Pin = static_cast<gpio_num_t>(doc["dht22Pin"].as<int>());
    config.dht22Interval = doc["dht22Interval"].as<int>();

    return true;
}
