#include "mqtt.h"
#include "main.h"
#include "wifi_manager.h"

WiFiClient wifiClient;
PubSubClient client(wifiClient);

TaskHandle_t mqttConnectTaskHandle;
TaskHandle_t mqttLoopTaskHandle;
TaskHandle_t healthPublisherTaskHandle;
TaskHandle_t waterPublisherTaskHandle;
QueueHandle_t waterPublisherQueue;
QueueHandle_t lightPublisherQueue;
TaskHandle_t lightPublisherTaskHandle;

char waterCommandTopic[50];
char stopCommandTopic[50];
char stopAllCommandTopic[50];
char waterDataTopic[50];
char lightCommandTopic[50];
char lightDataTopic[50];
char healthDataTopic[50];

#define ZERO (unsigned long int) 0

void setupMQTT() {
    // Connect to MQTT
    printf("connecting to mqtt server: %s:%d\n", mqtt_server, mqtt_port);
    client.setServer(mqtt_server, mqtt_port);
    client.setCallback(processIncomingMessage);
    client.setKeepAlive(MQTT_KEEPALIVE);

    snprintf(waterCommandTopic, sizeof(waterCommandTopic), "%s" MQTT_WATER_TOPIC, mqtt_topic_prefix);
    snprintf(stopCommandTopic, sizeof(stopCommandTopic), "%s" MQTT_STOP_TOPIC, mqtt_topic_prefix);
    snprintf(stopAllCommandTopic, sizeof(stopAllCommandTopic), "%s" MQTT_STOP_ALL_TOPIC, mqtt_topic_prefix);
    snprintf(waterDataTopic, sizeof(waterDataTopic), "%s" MQTT_WATER_DATA_TOPIC, mqtt_topic_prefix);
    snprintf(lightCommandTopic, sizeof(lightCommandTopic), "%s" MQTT_LIGHT_TOPIC, mqtt_topic_prefix);
    snprintf(lightDataTopic, sizeof(lightDataTopic), "%s" MQTT_LIGHT_DATA_TOPIC, mqtt_topic_prefix);
    snprintf(healthDataTopic, sizeof(healthDataTopic), "%s" MQTT_HEALTH_DATA_TOPIC, mqtt_topic_prefix);

    // Initialize publisher Queue
    waterPublisherQueue = xQueueCreate(QUEUE_SIZE, sizeof(WaterEvent));
    if (waterPublisherQueue == NULL) {
        printf("error creating the waterPublisherQueue\n");
    }

    // Start MQTT tasks
    xTaskCreate(mqttConnectTask, "MQTTConnectTask", 2048, NULL, 1, &mqttConnectTaskHandle);
    xTaskCreate(mqttLoopTask, "MQTTLoopTask", 4096, NULL, 1, &mqttLoopTaskHandle);
    xTaskCreate(waterPublisherTask, "WaterPublisherTask", 2048, NULL, 1, &waterPublisherTaskHandle);

    if (lightEnabled) {
        lightPublisherQueue = xQueueCreate(QUEUE_SIZE, sizeof(LightEvent));
        if (lightPublisherQueue == NULL) {
            printf("error creating the lightPublisherQueue\n");
        }
        xTaskCreate(lightPublisherTask, "LightPublisherTask", 2048, NULL, 1, &lightPublisherTaskHandle);
        xTaskCreate(healthPublisherTask, "HealthPublisherTask", 2048, NULL, 1, &healthPublisherTaskHandle);
    }
}

/*
  waterPublisherTask reads from a queue to publish WaterEvents as an InfluxDB
  line protocol message to MQTT
*/
void waterPublisherTask(void* parameters) {
    WaterEvent we;
    while (true) {
        if (xQueueReceive(waterPublisherQueue, &we, portMAX_DELAY)) {
            char message[50];
            sprintf(message, "water,zone=%d millis=%lu", we.position, we.duration);
            if (client.connected()) {
                printf("publishing to MQTT:\n\ttopic=%s\n\tmessage=%s\n", waterDataTopic, message);
                client.publish(waterDataTopic, message);
            } else {
                printf("unable to publish: not connected to MQTT broker\n");
            }
        }
        vTaskDelay(5 / portTICK_PERIOD_MS);
    }
    vTaskDelete(NULL);
}

/*
  lightPublisherTask reads from a queue to publish LightEvents as an InfluxDB
  line protocol message to MQTT
*/
void lightPublisherTask(void* parameters) {
    int state;
    while (true) {
        if (xQueueReceive(lightPublisherQueue, &state, portMAX_DELAY)) {
            char message[50];
            sprintf(message, "light,garden=\"%s\" state=%d", mqtt_topic_prefix, state);
            if (client.connected()) {
                printf("publishing to MQTT:\n\ttopic=%s\n\tmessage=%s\n", lightDataTopic, message);
                client.publish(lightDataTopic, message);
            } else {
                printf("unable to publish: not connected to MQTT broker\n");
            }
        }
        vTaskDelay(5 / portTICK_PERIOD_MS);
    }
    vTaskDelete(NULL);
}

/*
  healthPublisherTask runs every minute and publishes a message to MQTT to record a health check-in
*/
void healthPublisherTask(void* parameters) {
    WaterEvent we;
    while (true) {
        char message[50];
        sprintf(message, "health garden=\"%s\"", mqtt_topic_prefix);
        if (client.connected()) {
            printf("publishing to MQTT:\n\ttopic=%s\n\tmessage=%s\n", healthDataTopic, message);
            client.publish(healthDataTopic, message);
        } else {
            printf("unable to publish: not connected to MQTT broker\n");
        }
        vTaskDelay(HEALTH_PUBLISH_INTERVAL / portTICK_PERIOD_MS);
    }
    vTaskDelete(NULL);
}

/*
  mqttConnectTask will periodically attempt to reconnect to MQTT if needed
*/
void mqttConnectTask(void* parameters) {
    while (true) {
        // Connect to MQTT server if not connected already
        if (!client.connected()) {
            printf("attempting MQTT connection...");
            // Connect with defaul arguments + cleanSession = false for persistent sessions
            if (client.connect(mqtt_topic_prefix, NULL, NULL, 0, 0, 0, 0, false)) {
                printf("connected\n");
                client.subscribe(waterCommandTopic, 1);
                client.subscribe(stopCommandTopic, 1);
                client.subscribe(stopAllCommandTopic, 1);

                if (lightEnabled) {
                    client.subscribe(lightCommandTopic, 1);
                }
            } else {
                printf("failed, rc=%zu\n", client.state());
            }
        }
        vTaskDelay(5000 / portTICK_PERIOD_MS);
    }
    vTaskDelete(NULL);
}

/*
  mqttLoopTask will run the MQTT client loop to listen on subscribed topics
*/
void mqttLoopTask(void* parameters) {
    while (true) {
        // Run MQTT loop to process incoming messages if connected
        if (client.connected()) {
            client.loop();
        }
        vTaskDelay(5 / portTICK_PERIOD_MS);
    }
    vTaskDelete(NULL);
}

/*
  processIncomingMessage is a callback function for the MQTT client that will
  react to incoming messages. Currently, the topics are:
    - waterCommandTopic: accepts a WaterEvent JSON to water a zone for
                         specified time
    - stopCommandTopic: ignores message and stops the currently-watering zone
    - stopAllCommandTopic: ignores message, stops the currently-watering zone,
                           and clears the waterQueue
    - lightCommandTopic: accepts LightEvent JSON to control a grow light
*/
void processIncomingMessage(char* topic, byte* message, unsigned int length) {
    printf("message received:\n\ttopic=%s\n\tmessage=%s\n", topic, (char*)message);

    DynamicJsonDocument doc(1024);
    DeserializationError err = deserializeJson(doc, message);
    if (err) {
        printf("deserialize failed: %s\n", err.c_str());
    }

    if (strcmp(topic, waterCommandTopic) == 0) {
        WaterEvent we = {
            doc["position"] | -1,
            doc["duration"] | ZERO,
            doc["id"] | "N/A"
        };
        printf("received command to water zone %d (%s) for %lu\n", we.position, we.id, we.duration);
        waterZone(we);
    } else if (strcmp(topic, stopCommandTopic) == 0) {
        printf("received command to stop watering\n");
        stopWatering();
    } else if (strcmp(topic, stopAllCommandTopic) == 0) {
        printf("received command to stop ALL watering\n");
        stopAllWatering();
    } else if (strcmp(topic, lightCommandTopic) == 0) {
        LightEvent le = {
            doc["state"] | ""
        };
        printf("received command to change state of the light: '%s'\n", le.state);
        changeLight(le);
    }
}
