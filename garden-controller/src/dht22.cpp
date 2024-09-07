#include "config.h"

#include <Arduino.h>
#include "dht22.h"
#include "main.h"
#include "mqtt.h"
#include "DHT.h"
#include "wifi_manager.h"

TaskHandle_t dht22TaskHandle;

char temperatureDataTopic[50];
char humidityDataTopic[50];

DHT dht(DHT22_PIN, DHT22);

void setupDHT22() {
    snprintf(temperatureDataTopic, sizeof(temperatureDataTopic), "%s" MQTT_TEMPERATURE_DATA_TOPIC, mqtt_topic_prefix);
    snprintf(humidityDataTopic, sizeof(humidityDataTopic), "%s" MQTT_TEMPERATURE_DATA_TOPIC, mqtt_topic_prefix);

    dht.begin();
    xTaskCreate(dht22PublishTask, "DHT22Task", 2048, NULL, 1, &dht22TaskHandle);
}

void dht22PublishTask(void* parameters) {
    while (true) {
        float t = dht.readTemperature();
        float h = dht.readHumidity();

        printf("Temperature value: %f\n", t);
        printf("Humidity value: %f\n", h);

        char t_message[50];
        sprintf(t_message, "temperature value=%f", t);

        char h_message[50];
        sprintf(h_message, "humidity value=%f", h);

        if (client.connected()) {
            printf("publishing to MQTT:\n\ttopic=%s\n\tmessage=%s\n", temperatureDataTopic, t_message);
            client.publish(temperatureDataTopic, t_message);

            printf("publishing to MQTT:\n\ttopic=%s\n\tmessage=%s\n", humidityDataTopic, h_message);
            client.publish(humidityDataTopic, h_message);
        } else {
            printf("unable to publish: not connected to MQTT broker\n");
        }
        vTaskDelay(DHT22_INTERVAL / portTICK_PERIOD_MS);
    }
    vTaskDelete(NULL);
}
