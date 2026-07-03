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
        true, // fan
        GPIO_NUM_22, // fanPin
        2, // numSensors
        {
            { "", "DHT22", GPIO_NUM_21, 5000 },
            { "", "DS18B20", GPIO_NUM_22, 5000 }
        }
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
    TEST_ASSERT_EQUAL(inputConfig.fan, outputConfig.fan);
    TEST_ASSERT_EQUAL(inputConfig.fanPin, outputConfig.fanPin);

    TEST_ASSERT_EQUAL(inputConfig.numSensors, outputConfig.numSensors);
    for (int i = 0; i < inputConfig.numSensors; i++) {
        TEST_ASSERT_EQUAL_STRING(inputConfig.sensors[i].type, outputConfig.sensors[i].type);
        TEST_ASSERT_EQUAL(inputConfig.sensors[i].pin, outputConfig.sensors[i].pin);
        TEST_ASSERT_EQUAL(inputConfig.sensors[i].interval, outputConfig.sensors[i].interval);
    }
}

void test_serializeConfig(void) {
    Config inputConfig = {
        4, // numZones
        { GPIO_NUM_4, GPIO_NUM_5, GPIO_NUM_6, GPIO_NUM_7 }, // valvePins
        { GPIO_NUM_12, GPIO_NUM_13, GPIO_NUM_14, GPIO_NUM_15 }, // pumpPins
        true, // light
        GPIO_NUM_2, // lightPin
        true, // fan
        GPIO_NUM_22, // fanPin
        2, // numSensors
        {
            { "", "DHT22", GPIO_NUM_21, 5000 },
            { "", "DS18B20", GPIO_NUM_22, 5000 }
        }
    };

    String outputJSON;
    serializeConfig(inputConfig, outputJSON);

    TEST_ASSERT_EQUAL_STRING("{\"num_zones\":4,\"valve_pins\":[4,5,6,7],\"pump_pins\":[12,13,14,15],\"light\":true,\"light_pin\":2,\"fan\":true,\"fan_pin\":22,\"num_sensors\":2,\"sensors\":[{\"id\":\"\",\"type\":\"DHT22\",\"pin\":21,\"interval\":5000},{\"id\":\"\",\"type\":\"DS18B20\",\"pin\":22,\"interval\":5000}]}", outputJSON.c_str());
}

void test_deserializeConfig(void) {
    const char* inputJSON = "{\"num_zones\":4,\"valve_pins\":[4,5,6,7],\"pump_pins\":[12,13,14,15],\"light\":true,\"light_pin\":2,\"fan\":true,\"fan_pin\":22,\"num_sensors\":2,\"sensors\":[{\"type\":\"DHT22\",\"pin\":21,\"interval\":5000},{\"type\":\"DS18B20\",\"pin\":22,\"interval\":5000}]}";
    Config outputConfig;

    bool result = deserializeConfig(inputJSON, outputConfig);

    TEST_ASSERT_TRUE(result);

    Config expectedConfig = {
        4, // numZones
        { GPIO_NUM_4, GPIO_NUM_5, GPIO_NUM_6, GPIO_NUM_7 }, // valvePins
        { GPIO_NUM_12, GPIO_NUM_13, GPIO_NUM_14, GPIO_NUM_15 }, // pumpPins
        true, // light
        GPIO_NUM_2, // lightPin
        true, // fan
        GPIO_NUM_22, // fanPin
        2, // numSensors
        {
            { "", "DHT22", GPIO_NUM_21, 5000 },
            { "", "DS18B20", GPIO_NUM_22, 5000 }
        }
    };

    TEST_ASSERT_EQUAL(expectedConfig.numZones, outputConfig.numZones);

    for (int i = 0; i < expectedConfig.numZones; i++) {
        TEST_ASSERT_EQUAL(expectedConfig.valvePins[i], outputConfig.valvePins[i]);
        TEST_ASSERT_EQUAL(expectedConfig.pumpPins[i], outputConfig.pumpPins[i]);
    }

    TEST_ASSERT_EQUAL(expectedConfig.light, outputConfig.light);
    TEST_ASSERT_EQUAL(expectedConfig.lightPin, outputConfig.lightPin);
    TEST_ASSERT_EQUAL(expectedConfig.fan, outputConfig.fan);
    TEST_ASSERT_EQUAL(expectedConfig.fanPin, outputConfig.fanPin);

    TEST_ASSERT_EQUAL(expectedConfig.numSensors, outputConfig.numSensors);
    for (int i = 0; i < expectedConfig.numSensors; i++) {
        TEST_ASSERT_EQUAL_STRING(expectedConfig.sensors[i].type, outputConfig.sensors[i].type);
        TEST_ASSERT_EQUAL(expectedConfig.sensors[i].pin, outputConfig.sensors[i].pin);
        TEST_ASSERT_EQUAL(expectedConfig.sensors[i].interval, outputConfig.sensors[i].interval);
    }
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
