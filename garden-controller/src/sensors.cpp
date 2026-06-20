#include <Arduino.h>
#include <stdio.h>
#include <string.h>
#include <algorithm>

#include "sensors.h"
#include "main.h"
#include "mqtt.h"
#include "wifi_manager.h"
#include "DHT.h"
#include <OneWire.h>
#include <DallasTemperature.h>

#define MAX_SENSORS 16
#define MAX_DS18B20_BUSES 16
#define SENSOR_MESSAGE_SIZE 80
#define DHT22_MIN_INTERVAL_MS 2000
#define DS18B20_MIN_INTERVAL_MS 1000

TaskHandle_t sensorTaskHandle;

char sensorDataTopic[50];

struct DS18B20Bus {
    gpio_num_t pin;
    OneWire* oneWire;
    DallasTemperature* dt;
    int deviceCount;
    DeviceAddress addresses[16];
};

DS18B20Bus ds18b20Buses[MAX_DS18B20_BUSES];
int numDS18B20Buses = 0;

struct SensorRuntime {
    SensorConfig config;
    DHT* dht;
    DallasTemperature* dt;
    OneWire* oneWire;
    uint8_t ds18b20Index;
    unsigned long nextReadMillis;
};

SensorRuntime sensorRuntimes[MAX_SENSORS];
int numSensorRuntimes = 0;

static uint64_t addressToUint64(const DeviceAddress& addr) {
    uint64_t result = 0;
    for (int i = 0; i < 8; i++) {
        result = (result << 8) | addr[i];
    }
    return result;
}

static void sortAddresses(DeviceAddress addresses[], int count) {
    int indices[16];
    for (int i = 0; i < count; i++) {
        indices[i] = i;
    }
    std::sort(indices, indices + count, [&](int a, int b) {
        return addressToUint64(addresses[a]) < addressToUint64(addresses[b]);
    });
    DeviceAddress sorted[16];
    for (int i = 0; i < count; i++) {
        memcpy(sorted[i], addresses[indices[i]], 8);
    }
    for (int i = 0; i < count; i++) {
        memcpy(addresses[i], sorted[i], 8);
    }
}

static DS18B20Bus* findOrCreateBus(gpio_num_t pin) {
    for (int i = 0; i < numDS18B20Buses; i++) {
        if (ds18b20Buses[i].pin == pin) {
            return &ds18b20Buses[i];
        }
    }
    if (numDS18B20Buses >= MAX_DS18B20_BUSES) {
        printf("too many DS18B20 buses\n");
        return nullptr;
    }
    DS18B20Bus* bus = &ds18b20Buses[numDS18B20Buses++];
    bus->pin = pin;
    bus->oneWire = new OneWire(pin);
    bus->dt = new DallasTemperature(bus->oneWire);
    bus->dt->begin();
    bus->deviceCount = bus->dt->getDeviceCount();
    for (int i = 0; i < bus->deviceCount && i < 16; i++) {
        if (!bus->dt->getAddress(bus->addresses[i], i)) {
            printf("failed to get DS18B20 address for device %d on pin %d\n", i, (int)pin);
        }
    }
    sortAddresses(bus->addresses, bus->deviceCount);
    printf("DS18B20 bus on pin %d: found %d devices\n", (int)pin, bus->deviceCount);
    return bus;
}

static int clampInterval(const char* type, int interval) {
    if (interval <= 0) {
        return 5000;
    }
    if (strcasecmp(type, "DHT22") == 0 && interval < DHT22_MIN_INTERVAL_MS) {
        return DHT22_MIN_INTERVAL_MS;
    }
    if (strcasecmp(type, "DS18B20") == 0 && interval < DS18B20_MIN_INTERVAL_MS) {
        return DS18B20_MIN_INTERVAL_MS;
    }
    return interval;
}

void setupSensors() {
    printf("setting up sensors\n");
    snprintf(sensorDataTopic, sizeof(sensorDataTopic), "%s" MQTT_SENSOR_DATA_TOPIC, mqtt_topic_prefix);

    numSensorRuntimes = config.numSensors;
    if (numSensorRuntimes > MAX_SENSORS) {
        numSensorRuntimes = MAX_SENSORS;
    }

    for (int i = 0; i < numSensorRuntimes; i++) {
        sensorRuntimes[i].config = config.sensors[i];
        sensorRuntimes[i].config.interval = clampInterval(config.sensors[i].type, config.sensors[i].interval);
        sensorRuntimes[i].dht = nullptr;
        sensorRuntimes[i].dt = nullptr;
        sensorRuntimes[i].oneWire = nullptr;
        sensorRuntimes[i].ds18b20Index = 0;
        sensorRuntimes[i].nextReadMillis = millis();

        if (strcasecmp(config.sensors[i].type, "DHT22") == 0) {
            sensorRuntimes[i].dht = new DHT(config.sensors[i].pin, DHT22);
            sensorRuntimes[i].dht->begin();
            printf("sensor %d: DHT22 on pin %d interval=%d\n", i, (int)config.sensors[i].pin, sensorRuntimes[i].config.interval);
        } else if (strcasecmp(config.sensors[i].type, "DS18B20") == 0) {
            DS18B20Bus* bus = findOrCreateBus(config.sensors[i].pin);
            if (bus == nullptr) {
                printf("sensor %d: failed to create DS18B20 bus on pin %d\n", i, (int)config.sensors[i].pin);
                continue;
            }
            int usedCount = 0;
            for (int j = 0; j < i; j++) {
                if (strcasecmp(config.sensors[j].type, "DS18B20") == 0 && config.sensors[j].pin == config.sensors[i].pin) {
                    usedCount++;
                }
            }
            if (usedCount >= bus->deviceCount) {
                printf("sensor %d: not enough DS18B20 devices on pin %d (found %d)\n", i, (int)config.sensors[i].pin, bus->deviceCount);
                continue;
            }
            sensorRuntimes[i].dt = bus->dt;
            sensorRuntimes[i].oneWire = bus->oneWire;
            sensorRuntimes[i].ds18b20Index = usedCount;
            printf("sensor %d: DS18B20 on pin %d index %d interval=%d\n", i, (int)config.sensors[i].pin, usedCount, sensorRuntimes[i].config.interval);
        } else {
            printf("sensor %d: unknown type %s\n", i, config.sensors[i].type);
        }
    }

    if (numSensorRuntimes > 0) {
        xTaskCreate(sensorPublishTask, "SensorTask", 4096, NULL, 1, &sensorTaskHandle);
    }
}

static void publishSensor(int index, float temperature, float humidity, bool hasHumidity) {
    char message[SENSOR_MESSAGE_SIZE];
    if (hasHumidity) {
        snprintf(message, sizeof(message), "sensor,sensor_id=%d temperature=%.2f,humidity=%.2f", index, temperature, humidity);
    } else {
        snprintf(message, sizeof(message), "sensor,sensor_id=%d temperature=%.2f", index, temperature);
    }

    if (client.connected()) {
        printf("publishing to MQTT:\n\ttopic=%s\n\tmessage=%s\n", sensorDataTopic, message);
        client.publish(sensorDataTopic, message);
    } else {
        printf("unable to publish: not connected to MQTT broker\n");
    }
}

void sensorPublishTask(void* parameters) {
    while (true) {
        unsigned long now = millis();
        unsigned long nextWake = now + 1000;

        for (int i = 0; i < numSensorRuntimes; i++) {
            if ((long)(now - sensorRuntimes[i].nextReadMillis) >= 0) {
                if (sensorRuntimes[i].dht != nullptr) {
                    float t = sensorRuntimes[i].dht->readTemperature();
                    float h = sensorRuntimes[i].dht->readHumidity();
                    if (!isnan(t) && !isnan(h)) {
                        publishSensor(i, t, h, true);
                    } else {
                        printf("sensor %d: failed to read DHT22\n", i);
                    }
                } else if (sensorRuntimes[i].dt != nullptr) {
                    sensorRuntimes[i].dt->requestTemperatures();
                    float t = sensorRuntimes[i].dt->getTempCByIndex(sensorRuntimes[i].ds18b20Index);
                    if (t != DEVICE_DISCONNECTED_C) {
                        publishSensor(i, t, 0.0, false);
                    } else {
                        printf("sensor %d: failed to read DS18B20\n", i);
                    }
                }

                sensorRuntimes[i].nextReadMillis = now + sensorRuntimes[i].config.interval;
            }

            if ((long)(sensorRuntimes[i].nextReadMillis - nextWake) < 0) {
                nextWake = sensorRuntimes[i].nextReadMillis;
            }
        }

        long delayMs = (long)(nextWake - millis());
        if (delayMs > 0) {
            vTaskDelay(delayMs / portTICK_PERIOD_MS);
        } else {
            vTaskDelay(1);
        }
    }
    vTaskDelete(NULL);
}
