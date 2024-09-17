#ifndef GARDEN_CONFIG_H
#define GARDEN_CONFIG_H

#include <Arduino.h>
#include <ArduinoJson.h>

struct Config {
    int numZones;
    gpio_num_t valvePins[12];
    gpio_num_t pumpPins[12];

    bool light;
    gpio_num_t lightPin;

    bool tempHumidity;
    gpio_num_t tempHumidityPin;
    int tempHumidityInterval;
};

void serializeConfig(const Config& config, String& jsonString);
bool deserializeConfig(const char* jsonString, Config& config);
void initFS();
bool configFileExists();
void saveConfigToFile(const Config& config);
void loadConfigFromFile(Config& config);
void printConfig(Config& config);

#endif
