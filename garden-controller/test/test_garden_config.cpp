#include <Arduino.h>
#include <ArduinoJson.h>
#include <unity.h>
#include "garden_config.h"

void setUp(void) {}

void tearDown(void) {}

void test_loadAndSaveConfig() {
    Config inputConfig = {
        4, // numZones
        { GPIO_NUM_4, GPIO_NUM_5, GPIO_NUM_6, GPIO_NUM_7 }, // valvePins
        { GPIO_NUM_12, GPIO_NUM_13, GPIO_NUM_14, GPIO_NUM_15 }, // pumpPins
        true, // light
        GPIO_NUM_2, // lightPin
        true, // tempHumidity
        GPIO_NUM_21, // tempHumidityPin
        60 // tempHumidityInterval
    };

    initFS();
    saveConfigToFile(inputConfig);

    TEST_ASSERT_TRUE(configFileExists());

    Config outputConfig;
    loadConfigFromFile(outputConfig);

    TEST_ASSERT_EQUAL(inputConfig.numZones, outputConfig.numZones);

    for (int i = 0; i < inputConfig.numZones; i++) {
        TEST_ASSERT_EQUAL(inputConfig.valvePins[i], outputConfig.valvePins[i]);
        TEST_ASSERT_EQUAL(inputConfig.pumpPins[i], outputConfig.pumpPins[i]);
    }

    TEST_ASSERT_EQUAL(inputConfig.light, outputConfig.light);
    TEST_ASSERT_EQUAL(inputConfig.lightPin, outputConfig.lightPin);
    TEST_ASSERT_EQUAL(inputConfig.tempHumidity, outputConfig.tempHumidity);
    TEST_ASSERT_EQUAL(inputConfig.tempHumidityPin, outputConfig.tempHumidityPin);
    TEST_ASSERT_EQUAL(inputConfig.tempHumidityInterval, outputConfig.tempHumidityInterval);
}

void test_serializeConfig(void) {
    Config inputConfig = {
        4, // numZones
        { GPIO_NUM_4, GPIO_NUM_5, GPIO_NUM_6, GPIO_NUM_7 }, // valvePins
        { GPIO_NUM_12, GPIO_NUM_13, GPIO_NUM_14, GPIO_NUM_15 }, // pumpPins
        true, // light
        GPIO_NUM_2, // lightPin
        true, // tempHumidity
        GPIO_NUM_21, // tempHumidityPin
        60 // tempHumidityInterval
    };

    String outputJSON;
    serializeConfig(inputConfig, outputJSON);

    TEST_ASSERT_EQUAL_STRING("{\"num_zones\":4,\"valve_pins\":[4,5,6,7],\"pump_pins\":[12,13,14,15],\"light\":true,\"light_pin\":2,\"temp_humidity\":true,\"temp_humidity_pin\":21,\"temp_humidity_interval\":60}", outputJSON.c_str());
}

void test_deserializeConfig(void) {
    const char* inputJSON = "{\"num_zones\":4,\"valve_pins\":[4,5,6,7],\"pump_pins\":[12,13,14,15],\"light\":true,\"light_pin\":2,\"temp_humidity\":true,\"temp_humidity_pin\":21,\"temp_humidity_interval\":60}";
    Config outputConfig;

    bool result = deserializeConfig(inputJSON, outputConfig);

    TEST_ASSERT_TRUE(result);

    Config expectedConfig = {
        4, // numZones
        { GPIO_NUM_4, GPIO_NUM_5, GPIO_NUM_6, GPIO_NUM_7 }, // valvePins
        { GPIO_NUM_12, GPIO_NUM_13, GPIO_NUM_14, GPIO_NUM_15 }, // pumpPins
        true, // light
        GPIO_NUM_2, // lightPin
        true, // tempHumidity
        GPIO_NUM_21, // tempHumidityPin
        60 // tempHumidityInterval
    };

    TEST_ASSERT_EQUAL(expectedConfig.numZones, outputConfig.numZones);

    for (int i = 0; i < expectedConfig.numZones; i++) {
        TEST_ASSERT_EQUAL(expectedConfig.valvePins[i], outputConfig.valvePins[i]);
        TEST_ASSERT_EQUAL(expectedConfig.pumpPins[i], outputConfig.pumpPins[i]);
    }

    TEST_ASSERT_EQUAL(expectedConfig.light, outputConfig.light);
    TEST_ASSERT_EQUAL(expectedConfig.lightPin, outputConfig.lightPin);
    TEST_ASSERT_EQUAL(expectedConfig.tempHumidity, outputConfig.tempHumidity);
    TEST_ASSERT_EQUAL(expectedConfig.tempHumidityPin, outputConfig.tempHumidityPin);
    TEST_ASSERT_EQUAL(expectedConfig.tempHumidityInterval, outputConfig.tempHumidityInterval);
}

void setup() {
    Serial.begin(115200);

    UNITY_BEGIN();
    RUN_TEST(test_loadAndSaveConfig);
    RUN_TEST(test_serializeConfig);
    RUN_TEST(test_deserializeConfig);
    UNITY_END();
}

void loop() {}
