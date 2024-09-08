#include <Arduino.h>
#include <ArduinoJson.h>
#include <unity.h>
#include "garden_config.h"

void setUp(void) {}

void tearDown(void) {}

void test_loadAndSaveConfig() {
    Config inputConfig = {
        1883,
        "mqtt.example.com",
        "topic_prefix",
        4, // numZones
        { GPIO_NUM_4, GPIO_NUM_5, GPIO_NUM_6, GPIO_NUM_7 }, // zonePins
        { GPIO_NUM_12, GPIO_NUM_13, GPIO_NUM_14, GPIO_NUM_15 }, // pumpPins
        true, // light
        GPIO_NUM_2, // lightPin
        true, // dht22
        GPIO_NUM_21, // dht22Pin
        60 // dht22Interval
    };

    initFS();
    saveConfigToFile(inputConfig);

    TEST_ASSERT_TRUE(configFileExists());

    Config outputConfig;
    loadConfigFromFile(outputConfig);

    TEST_ASSERT_EQUAL(inputConfig.mqttPort, outputConfig.mqttPort);
    TEST_ASSERT_EQUAL_STRING(inputConfig.mqttServer, outputConfig.mqttServer);
    TEST_ASSERT_EQUAL_STRING(inputConfig.mqttTopicPrefix, outputConfig.mqttTopicPrefix);
    TEST_ASSERT_EQUAL(inputConfig.numZones, outputConfig.numZones);

    for (int i = 0; i < inputConfig.numZones; i++) {
        TEST_ASSERT_EQUAL(inputConfig.zonePins[i], outputConfig.zonePins[i]);
        TEST_ASSERT_EQUAL(inputConfig.pumpPins[i], outputConfig.pumpPins[i]);
    }

    TEST_ASSERT_EQUAL(inputConfig.light, outputConfig.light);
    TEST_ASSERT_EQUAL(inputConfig.lightPin, outputConfig.lightPin);
    TEST_ASSERT_EQUAL(inputConfig.dht22, outputConfig.dht22);
    TEST_ASSERT_EQUAL(inputConfig.dht22Pin, outputConfig.dht22Pin);
    TEST_ASSERT_EQUAL(inputConfig.dht22Interval, outputConfig.dht22Interval);
}

void test_serializeConfig(void) {
    Config inputConfig = {
        1883,
        "mqtt.example.com",
        "topic_prefix",
        4, // numZones
        { GPIO_NUM_4, GPIO_NUM_5, GPIO_NUM_6, GPIO_NUM_7 }, // zonePins
        { GPIO_NUM_12, GPIO_NUM_13, GPIO_NUM_14, GPIO_NUM_15 }, // pumpPins
        true, // light
        GPIO_NUM_2, // lightPin
        true, // dht22
        GPIO_NUM_21, // dht22Pin
        60 // dht22Interval
    };

    String outputJSON;
    serializeConfig(inputConfig, outputJSON);

    TEST_ASSERT_EQUAL_STRING("{\"mqttPort\":1883,\"mqttServer\":\"mqtt.example.com\",\"mqttTopicPrefix\":\"topic_prefix\",\"numZones\":4,\"zonePins\":[4,5,6,7],\"pumpPins\":[12,13,14,15],\"light\":true,\"lightPin\":2,\"dht22\":true,\"dht22Pin\":21,\"dht22Interval\":60}", outputJSON.c_str());
}

void test_deserializeConfig(void) {
    String inputJSON = "{\"mqttPort\":1883,\"mqttServer\":\"mqtt.example.com\",\"mqttTopicPrefix\":\"topic_prefix\",\"numZones\":4,\"zonePins\":[4,5,6,7],\"pumpPins\":[12,13,14,15],\"light\":true,\"lightPin\":2,\"dht22\":true,\"dht22Pin\":21,\"dht22Interval\":60}";
    Config outputConfig;

    bool result = deserializeConfig(inputJSON, outputConfig);

    TEST_ASSERT_TRUE(result);

    Config expectedConfig = {
        1883,
        "mqtt.example.com",
        "topic_prefix",
        4, // numZones
        { GPIO_NUM_4, GPIO_NUM_5, GPIO_NUM_6, GPIO_NUM_7 }, // zonePins
        { GPIO_NUM_12, GPIO_NUM_13, GPIO_NUM_14, GPIO_NUM_15 }, // pumpPins
        true, // light
        GPIO_NUM_2, // lightPin
        true, // dht22
        GPIO_NUM_21, // dht22Pin
        60 // dht22Interval
    };

    TEST_ASSERT_EQUAL(expectedConfig.mqttPort, outputConfig.mqttPort);
    TEST_ASSERT_EQUAL_STRING(expectedConfig.mqttServer, outputConfig.mqttServer);
    TEST_ASSERT_EQUAL_STRING(expectedConfig.mqttTopicPrefix, outputConfig.mqttTopicPrefix);
    TEST_ASSERT_EQUAL(expectedConfig.numZones, outputConfig.numZones);

    for (int i = 0; i < expectedConfig.numZones; i++) {
        TEST_ASSERT_EQUAL(expectedConfig.zonePins[i], outputConfig.zonePins[i]);
        TEST_ASSERT_EQUAL(expectedConfig.pumpPins[i], outputConfig.pumpPins[i]);
    }

    TEST_ASSERT_EQUAL(expectedConfig.light, outputConfig.light);
    TEST_ASSERT_EQUAL(expectedConfig.lightPin, outputConfig.lightPin);
    TEST_ASSERT_EQUAL(expectedConfig.dht22, outputConfig.dht22);
    TEST_ASSERT_EQUAL(expectedConfig.dht22Pin, outputConfig.dht22Pin);
    TEST_ASSERT_EQUAL(expectedConfig.dht22Interval, outputConfig.dht22Interval);
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
