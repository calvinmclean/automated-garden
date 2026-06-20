#ifndef GARDEN_CONFIG_H
#define GARDEN_CONFIG_H

#include <Arduino.h>
#include <ArduinoJson.h>

struct SensorConfig {
    char type[16];
    gpio_num_t pin;
    int interval;
};

struct Config {
    int numZones;
    gpio_num_t valvePins[12];
    gpio_num_t pumpPins[12];

    bool light;
    gpio_num_t lightPin;

    bool fan;
    gpio_num_t fanPin;

    int numSensors;
    SensorConfig sensors[16];
};

void serializeConfig(const Config& config, String& jsonString);
bool deserializeConfig(const char* jsonString, Config& config);
void initFS();
bool configFileExists();
void saveConfigToFile(const Config& config);
void loadConfigFromFile(Config& config);
void printConfig(Config& config);

#endif
