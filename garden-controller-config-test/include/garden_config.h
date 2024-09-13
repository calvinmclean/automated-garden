#ifndef GARDEN_CONFIG_H
#define GARDEN_CONFIG_H

#include <Arduino.h>
#include <ArduinoJson.h>

#define FORMAT_LITTLEFS_IF_FAILED true

struct Config {
    int mqttPort;
    const char* mqttServer;
    const char* mqttTopicPrefix;

    int numZones;
    gpio_num_t zonePins[12];
    gpio_num_t pumpPins[12];

    bool light;
    gpio_num_t lightPin;

    bool dht22;
    gpio_num_t dht22Pin;
    int dht22Interval;
};

void serializeConfig(const Config& config, String& jsonString);
bool deserializeConfig(const String& jsonString, Config& config);
void initFS();
bool configFileExists();
void saveConfigToFile(const Config& config);
void loadConfigFromFile(Config& config);

#endif
