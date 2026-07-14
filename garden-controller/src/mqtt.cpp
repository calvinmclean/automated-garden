#include "mqtt.h"
#include "main.h"
#include "wifi_manager.h"
#include "controller_info.h"

#include <esp_system.h>

WiFiClient wifiClient;
PubSubClient client(wifiClient);

SemaphoreHandle_t mqttMutex;
QueueHandle_t mqttCommandQueue;
TaskHandle_t mqttCommandTaskHandle;

TaskHandle_t mqttConnectTaskHandle;
TaskHandle_t mqttLoopTaskHandle;

TaskHandle_t healthPublisherTaskHandle;

TaskHandle_t waterPublisherTaskHandle;
QueueHandle_t waterPublisherQueue;

QueueHandle_t lightPublisherQueue;
TaskHandle_t lightPublisherTaskHandle;

QueueHandle_t fanPublisherQueue;
TaskHandle_t fanPublisherTaskHandle;

// command topics (subscribe)
char waterCommandTopic[80];
char stopCommandTopic[80];
char stopAllCommandTopic[80];
char lightCommandTopic[80];
char fanCommandTopic[80];
char updateConfigCommandTopic[80];

// data topics (publish)
char waterDataTopic[80];
char lightDataTopic[80];
char fanDataTopic[80];
char healthDataTopic[80];
char logDataTopic[80];
char infoDataTopic[80];

#define ZERO (unsigned long int) 0
#define MQTT_COMMAND_QUEUE_SIZE 32

struct MQTTCommand {
    char topic[80];
    char* message;
};

void mqttLock() {
    xSemaphoreTakeRecursive(mqttMutex, portMAX_DELAY);
}

void mqttUnlock() {
    xSemaphoreGiveRecursive(mqttMutex);
}

void setupMQTTMutexAndQueue() {
    mqttMutex = xSemaphoreCreateRecursiveMutex();
    if (mqttMutex == NULL) {
        printf("error creating the mqttMutex\n");
    }

    mqttCommandQueue = xQueueCreate(MQTT_COMMAND_QUEUE_SIZE, sizeof(MQTTCommand*));
    if (mqttCommandQueue == NULL) {
        printf("error creating the mqttCommandQueue\n");
    }

    xTaskCreate(mqttCommandTask, "MQTTCommandTask", 4096, NULL, 1, &mqttCommandTaskHandle);
}

void setupMQTT() {
    // Connect to MQTT
    printf("connecting to mqtt server: %s:%d\n", mqtt_server, mqtt_port);
    client.setServer(mqtt_server, mqtt_port);
    client.setCallback(processIncomingMessage);
    client.setKeepAlive(MQTT_KEEPALIVE);

    snprintf(waterCommandTopic, sizeof(waterCommandTopic), "%s" MQTT_WATER_TOPIC, mqtt_topic_prefix);
    snprintf(stopCommandTopic, sizeof(stopCommandTopic), "%s" MQTT_STOP_TOPIC, mqtt_topic_prefix);
    snprintf(stopAllCommandTopic, sizeof(stopAllCommandTopic), "%s" MQTT_STOP_ALL_TOPIC, mqtt_topic_prefix);
    snprintf(lightCommandTopic, sizeof(lightCommandTopic), "%s" MQTT_LIGHT_TOPIC, mqtt_topic_prefix);
    snprintf(fanCommandTopic, sizeof(fanCommandTopic), "%s" MQTT_FAN_TOPIC, mqtt_topic_prefix);
    snprintf(updateConfigCommandTopic, sizeof(updateConfigCommandTopic), "%s" MQTT_UPDATE_CONFIG_TOPIC, mqtt_topic_prefix);

    snprintf(waterDataTopic, sizeof(waterDataTopic), "%s" MQTT_WATER_DATA_TOPIC, mqtt_topic_prefix);
    snprintf(lightDataTopic, sizeof(lightDataTopic), "%s" MQTT_LIGHT_DATA_TOPIC, mqtt_topic_prefix);
    snprintf(fanDataTopic, sizeof(fanDataTopic), "%s" MQTT_FAN_DATA_TOPIC, mqtt_topic_prefix);
    snprintf(healthDataTopic, sizeof(healthDataTopic), "%s" MQTT_HEALTH_DATA_TOPIC, mqtt_topic_prefix);
    snprintf(logDataTopic, sizeof(logDataTopic), "%s" MQTT_LOGGING_TOPIC, mqtt_topic_prefix);
    snprintf(infoDataTopic, sizeof(infoDataTopic), "%s" MQTT_INFO_DATA_TOPIC, mqtt_topic_prefix);

    // printf("Topics:\n");
    // printf("  %s\n  %s\n  %s\n  %s\n  %s\n  %s\n  %s\n  %s\n  %s\n", waterCommandTopic,stopCommandTopic,stopAllCommandTopic,lightCommandTopic,updateConfigCommandTopic,waterDataTopic,lightDataTopic,healthDataTopic,logDataTopic);

    // Initialize publisher Queue
    waterPublisherQueue = xQueueCreate(QUEUE_SIZE, sizeof(WaterStatusEvent));
    if (waterPublisherQueue == NULL) {
        printf("error creating the waterPublisherQueue\n");
    }

    // Start MQTT tasks
    xTaskCreate(mqttConnectTask, "MQTTConnectTask", 4096, NULL, 1, &mqttConnectTaskHandle);
    xTaskCreate(mqttLoopTask, "MQTTLoopTask", 4096, NULL, 2, &mqttLoopTaskHandle);
    xTaskCreate(waterPublisherTask, "WaterPublisherTask", 2048, NULL, 1, &waterPublisherTaskHandle);
    xTaskCreate(healthPublisherTask, "HealthPublisherTask", 2048, NULL, 1, &healthPublisherTaskHandle);

    if (config.light) {
        lightPublisherQueue = xQueueCreate(QUEUE_SIZE, sizeof(LightEvent));
        if (lightPublisherQueue == NULL) {
            printf("error creating the lightPublisherQueue\n");
        }
        xTaskCreate(lightPublisherTask, "LightPublisherTask", 2048, NULL, 1, &lightPublisherTaskHandle);
    }

    if (config.fan) {
        fanPublisherQueue = xQueueCreate(QUEUE_SIZE, sizeof(int));
        if (fanPublisherQueue == NULL) {
            printf("error creating the fanPublisherQueue\n");
        }
        xTaskCreate(fanPublisherTask, "FanPublisherTask", 2048, NULL, 1, &fanPublisherTaskHandle);
    }
}

/*
  waterPublisherTask reads from a queue to publish WaterStatusEvents as an InfluxDB
  line protocol message to MQTT, then frees the heap-allocated strings that
  were transferred by the producer.
*/
void waterPublisherTask(void* parameters) {
    WaterStatusEvent we;
    char message[150];

    while (true) {
        if (xQueueReceive(waterPublisherQueue, &we, portMAX_DELAY)) {
            memset(message, '\0', sizeof(message));
            const char* statusStr;
            unsigned long millisVal;
            switch (we.status) {
                case WATER_START:
                    statusStr = "start";
                    millisVal = 0;
                    break;
                case WATER_COMPLETE:
                    statusStr = "complete";
                    millisVal = we.duration;
                    break;
                case WATER_CANCELLED:
                    statusStr = "cancelled";
                    millisVal = we.duration;
                    break;
            }
            snprintf(message, sizeof(message), "water,status=%s,zone=%d,id=%s,zone_id=%s millis=%lu",
                     statusStr, we.position, we.id, we.zone_id, millisVal);

            mqttLock();
            if (client.connected()) {
                printf("publishing to MQTT:\n\ttopic=%s\n\tmessage=%s\n", waterDataTopic, message);
                client.publish(waterDataTopic, message);
            } else {
                printf("unable to publish: not connected to MQTT broker\n");
            }
            mqttUnlock();

            free(we.zone_id);
            free(we.id);
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
            mqttLock();
            if (client.connected()) {
                printf("publishing to MQTT:\n\ttopic=%s\n\tmessage=%s\n", lightDataTopic, message);
                client.publish(lightDataTopic, message);
            } else {
                printf("unable to publish: not connected to MQTT broker\n");
            }
            mqttUnlock();
        }
        vTaskDelay(5 / portTICK_PERIOD_MS);
    }
    vTaskDelete(NULL);
}

/*
  fanPublisherTask reads from a queue to publish fan state as an InfluxDB
  line protocol message to MQTT
*/
void fanPublisherTask(void* parameters) {
    int power;
    while (true) {
        if (xQueueReceive(fanPublisherQueue, &power, portMAX_DELAY)) {
            char message[50];
            sprintf(message, "fan,garden=\"%s\" power=%d", mqtt_topic_prefix, power);
            mqttLock();
            if (client.connected()) {
                printf("publishing to MQTT:\n\ttopic=%s\n\tmessage=%s\n", fanDataTopic, message);
                client.publish(fanDataTopic, message);
            } else {
                printf("unable to publish: not connected to MQTT broker\n");
            }
            mqttUnlock();
        }
        vTaskDelay(5 / portTICK_PERIOD_MS);
    }
    vTaskDelete(NULL);
}

/*
  healthPublisherTask runs every minute and publishes a message to MQTT to record a health check-in
*/
void healthPublisherTask(void* parameters) {
    WaterMessage we;
    while (true) {
        char message[50];
        sprintf(message, "health garden=\"%s\"", mqtt_topic_prefix);
        mqttLock();
        if (client.connected()) {
            printf("publishing to MQTT:\n\ttopic=%s\n\tmessage=%s\n", healthDataTopic, message);
            client.publish(healthDataTopic, message);
        } else {
            printf("unable to publish: not connected to MQTT broker\n");
        }
        mqttUnlock();
        vTaskDelay(HEALTH_PUBLISH_INTERVAL / portTICK_PERIOD_MS);
    }
    vTaskDelete(NULL);
}

// resetReasonString returns a human-readable string for the ESP32 reset reason.
const char* resetReasonString(esp_reset_reason_t reason) {
    switch (reason) {
        case ESP_RST_UNKNOWN:
            return "Reset reason can not be determined.";
        case ESP_RST_POWERON:
            return "Reset due to power-on event.";
        case ESP_RST_EXT:
            return "Reset by external pin (not applicable for ESP32)";
        case ESP_RST_SW:
            return "Software reset via esp_restart.";
        case ESP_RST_PANIC:
            return "Software reset due to exception/panic.";
        case ESP_RST_INT_WDT:
            return "Reset (software or hardware) due to interrupt watchdog.";
        case ESP_RST_TASK_WDT:
            return "Reset due to task watchdog.";
        case ESP_RST_WDT:
            return "Reset due to other watchdogs.";
        case ESP_RST_DEEPSLEEP:
            return "Reset after exiting deep sleep mode.";
        case ESP_RST_BROWNOUT:
            return "Brownout reset (software or hardware)";
        case ESP_RST_SDIO:
            return "Reset over SDIO.";
#ifdef ESP_RST_USB
        case ESP_RST_USB:
            return "Reset by USB peripheral.";
#endif
#ifdef ESP_RST_JTAG
        case ESP_RST_JTAG:
            return "Reset by JTAG.";
#endif
#ifdef ESP_RST_EFUSE
        case ESP_RST_EFUSE:
            return "Reset due to efuse error.";
#endif
#ifdef ESP_RST_PWR_GLITCH
        case ESP_RST_PWR_GLITCH:
            return "Reset due to power glitch detected.";
#endif
#ifdef ESP_RST_CPU_LOCKUP
        case ESP_RST_CPU_LOCKUP:
            return "Reset due to CPU lock up (double exception)";
#endif
        default:
            return "Reset reason can not be determined.";
    }
}

/*
  mqttConnectTask will periodically attempt to reconnect to MQTT if needed
*/
void mqttConnectTask(void* parameters) {
    static bool firstConnect = true;
    while (true) {
        // Connect to MQTT server if not connected already
        mqttLock();
        if (!client.connected()) {
            printf("attempting MQTT connection...");
            // Connect with defaul arguments + cleanSession = false for persistent sessions
            if (client.connect(mqtt_topic_prefix, NULL, NULL, 0, 0, 0, 0, false)) {
                printf(firstConnect ? "connected\n" : "reconnected\n");
                client.subscribe(waterCommandTopic, 1);
                client.subscribe(stopCommandTopic, 1);
                client.subscribe(stopAllCommandTopic, 1);
                client.subscribe(updateConfigCommandTopic, 1);

                if (config.light) {
                    client.subscribe(lightCommandTopic, 1);
                }

                if (config.fan) {
                    client.subscribe(fanCommandTopic, 1);
                }

                if (firstConnect) {
                    publishLog("info", "startup", "garden-controller setup complete",
                               {{"reset_reason", resetReasonString(esp_reset_reason())}});
                    firstConnect = false;
                }
                publishControllerInfo();
            } else {
                printf("failed, rc=%zu\n", client.state());
            }
        }
        mqttUnlock();
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
        mqttLock();
        if (client.connected()) {
            client.loop();
        }
        mqttUnlock();
        vTaskDelay(5 / portTICK_PERIOD_MS);
    }
    vTaskDelete(NULL);
}

void handleWaterCommand(char* message) {
    DynamicJsonDocument doc(1024);
    DeserializationError err = deserializeJson(doc, message);
    if (err) {
        printf("deserialize failed: %s\n", err.c_str());
    }

    WaterMessage we = {
        doc["position"] | -1,
        doc["duration"] | ZERO,
        strdup(doc["zone_id"] | "N/A"),
        strdup(doc["id"] | "N/A"),
        WATER_START
    };
    waterZone(we);
}

void handleLightCommand(char* message) {
    DynamicJsonDocument doc(1024);
    DeserializationError err = deserializeJson(doc, message);
    if (err) {
        printf("deserialize failed: %s\n", err.c_str());
    }

    LightEvent le = {
        doc["state"] | ""
    };
    printf("received command to change state of the light: '%s'\n", le.state);
    changeLight(le);
}

void handleFanCommand(char* message) {
    DynamicJsonDocument doc(1024);
    DeserializationError err = deserializeJson(doc, message);
    if (err) {
        printf("deserialize failed: %s\n", err.c_str());
    }

    FanEvent fe = {
        doc["duration"] | ZERO,
        (unsigned int)(doc["power"] | 0)
    };
    printf("received command to run fan for %lu at power %d\n", fe.duration, fe.power);
    changeFan(fe);
}

void handleConfigCommand(char* message) {
    printf("handling update_config command\n");
    bool result = deserializeConfig((char*)message, config);
    if (!result) {
        printf("failed to deserialize config: %s\n", (char*)message);
    } else {
        printf("config deserialized successfully, numSensors=%d\n", config.numSensors);
    }

    saveConfigToFile(config);

    reboot(1000);
}

void publishInfoMessage(const char* message) {
    mqttLock();
    if (client.connected()) {
        printf("publishing to MQTT:\n\ttopic=%s\n\tmessage=%s\n", infoDataTopic, message);
        client.publish(infoDataTopic, message);
    } else {
        printf("unable to publish controller info: not connected to MQTT broker\n");
    }
    mqttUnlock();
}

// escapeLineProtocolString copies src into dest, escaping quotes and backslashes,
// and returns the number of bytes written (excluding the null terminator).
static size_t escapeLineProtocolString(const char* src, char* dest, size_t destSize) {
    size_t j = 0;
    for (size_t i = 0; src[i] != '\0' && j < destSize - 2; i++) {
        if (src[i] == '"' || src[i] == '\\') {
            dest[j++] = '\\';
        }
        dest[j++] = src[i];
    }
    dest[j] = '\0';
    return j;
}

/*
  publishLog publishes a generic log message to the logging topic. The message
  is formatted as InfluxDB line protocol with level and source tags and a
  string message field. The message content and any extra field values have
  basic escaping for quotes and backslashes.
*/
void publishLog(const char* level, const char* source, const char* message,
                std::initializer_list<std::pair<const char*, const char*>> extraFields) {
    mqttLock();
    if (client.connected()) {
        char escapedMessage[256];
        escapeLineProtocolString(message, escapedMessage, sizeof(escapedMessage));

        char extraFieldString[256];
        extraFieldString[0] = '\0';
        size_t pos = 0;
        for (const auto& kv : extraFields) {
            if (kv.first == nullptr || kv.second == nullptr) {
                continue;
            }
            int written = snprintf(extraFieldString + pos, sizeof(extraFieldString) - pos, ",%s=\"", kv.first);
            if (written < 0 || (size_t)written >= sizeof(extraFieldString) - pos) {
                break;
            }
            pos += written;
            pos += escapeLineProtocolString(kv.second, extraFieldString + pos, sizeof(extraFieldString) - pos);
            if (pos < sizeof(extraFieldString) - 1) {
                extraFieldString[pos++] = '"';
                extraFieldString[pos] = '\0';
            }
        }

        char formattedMessage[512];
        snprintf(formattedMessage, sizeof(formattedMessage),
                 "logs,level=%s,source=%s message=\"%s\"%s", level, source, escapedMessage, extraFieldString);

        printf("publishing to MQTT:\n\ttopic=%s\n\tmessage=%s\n", logDataTopic, formattedMessage);
        client.publish(logDataTopic, formattedMessage);
    } else {
        printf("unable to publish log: not connected to MQTT broker\n");
    }
    mqttUnlock();
}

/*
  processIncomingMessage is a callback function for the MQTT client that will
  react to incoming messages. Currently, the topics are:
    - waterCommandTopic: accepts a WaterMessage JSON to water a zone for
                         specified time
    - stopCommandTopic: ignores message and stops the currently-watering zone
    - stopAllCommandTopic: ignores message, stops the currently-watering zone,
                           and clears the waterQueue
    - lightCommandTopic: accepts LightEvent JSON to control a grow light
    - fanCommandTopic: accepts FanEvent JSON to control a fan
    - updateConfigCommandTopic: accepts Config JSON to update
*/
void processIncomingMessage(char* topic, byte* message, unsigned int length) {
    if (length == 0) {
        return;
    }

    MQTTCommand* cmd = (MQTTCommand*)malloc(sizeof(MQTTCommand));
    if (cmd == nullptr) {
        printf("memory allocation failed for MQTT command\n");
        return;
    }

    cmd->message = (char*)malloc(length + 1);
    if (cmd->message == nullptr) {
        printf("memory allocation failed for MQTT command message\n");
        free(cmd);
        return;
    }

    strncpy(cmd->topic, topic, sizeof(cmd->topic) - 1);
    cmd->topic[sizeof(cmd->topic) - 1] = '\0';
    memcpy(cmd->message, message, length);
    cmd->message[length] = '\0';

    if (xQueueSend(mqttCommandQueue, &cmd, 0) != pdPASS) {
        printf("MQTT command queue full, dropping message\n");
        free(cmd->message);
        free(cmd);
    }
}

void mqttCommandTask(void* parameters) {
    MQTTCommand* cmd;
    while (true) {
        if (xQueueReceive(mqttCommandQueue, &cmd, portMAX_DELAY)) {
            printf("message received:\n\ttopic=%s\n\tmessage=%s\n", cmd->topic, cmd->message);

            if (strcmp(cmd->topic, waterCommandTopic) == 0) {
                handleWaterCommand(cmd->message);
            } else if (strcmp(cmd->topic, stopCommandTopic) == 0) {
                printf("received command to stop watering\n");
                stopWatering();
            } else if (strcmp(cmd->topic, stopAllCommandTopic) == 0) {
                printf("received command to stop ALL watering\n");
                stopAllWatering();
            } else if (strcmp(cmd->topic, lightCommandTopic) == 0) {
                handleLightCommand(cmd->message);
            } else if (strcmp(cmd->topic, fanCommandTopic) == 0) {
                handleFanCommand(cmd->message);
            } else if (strcmp(cmd->topic, updateConfigCommandTopic) == 0) {
                handleConfigCommand(cmd->message);
            } else {
                printf("unexpected topic: %s\n", cmd->topic);
            }

            free(cmd->message);
            free(cmd);
        }
    }
    vTaskDelete(NULL);
}
