#include <ArduinoJson.h>
#include <PubSubClient.h>
#include <stdio.h>
#include "driver/gpio.h"

/* include other files for this program */
#include "wifi.h"
#include "config.h"

#define JSON_CAPACITY 128
#define QUEUE_SIZE 10

#define INTERVAL 86400000 // 24 hours
#define DEFAULT_WATER_TIME 15000

#define DEBOUNCE_DELAY 50

#define MQTT_RETRY_DELAY 5000
#define MQTT_CLIENT_NAME "Garden"

typedef struct WateringEvent {
    int plant_position;
    unsigned long duration;
    const char* id;
};

/* plant/valve variables */
gpio_num_t plants[NUM_PLANTS][3] = PLANTS;
bool skipValve[NUM_PLANTS] = { false, false, false };

/* button variables */
unsigned long lastDebounceTime = 0;
int buttonStates[NUM_PLANTS] = { LOW, LOW, LOW };
int lastButtonStates[NUM_PLANTS] = { LOW, LOW, LOW };

/* stop button variables */
unsigned long lastStopDebounceTime = 0;
int stopButtonState = LOW;
int lastStopButtonState;

/* watering cycle variables */
unsigned long previousMillis = -INTERVAL;
int watering = -1;

/* MQTT variables */
unsigned long lastConnectAttempt = 0;
PubSubClient client(wifiClient);
const char* waterCommandTopic = "garden/command/water";
const char* stopCommandTopic = "garden/command/stop";
const char* stopAllCommandTopic = "garden/command/stop_all";
const char* skipCommandTopic = "garden/command/skip";
const char* waterDataTopic = "garden/data/water";

/* FreeRTOS Queue and Task handlers */
QueueHandle_t publisherQueue;
QueueHandle_t wateringQueue;
TaskHandle_t mqttConnectTaskHandle;
TaskHandle_t mqttLoopTaskHandle;
TaskHandle_t publisherTaskHandle;
TaskHandle_t waterPlantTaskHandle;
TaskHandle_t waterIntervalTaskHandle;
TaskHandle_t readButtonsTaskHandle;

void setup() {
    // Prepare pins
    for (int i = 0; i < NUM_PLANTS; i++) {
        // Setup button pins
        gpio_reset_pin(plants[i][2]);
        gpio_set_direction(plants[i][2], GPIO_MODE_INPUT);

        // Setup valve pins
        gpio_reset_pin(plants[i][1]);
        gpio_set_direction(plants[i][1], GPIO_MODE_OUTPUT);

        // Setup pump pins
        gpio_reset_pin(plants[i][0]);
        gpio_set_direction(plants[i][0], GPIO_MODE_OUTPUT);
    }

    // Connect to WiFi and MQTT
    setup_wifi();
    client.setServer(MQTT_ADDRESS, MQTT_PORT);
    client.setCallback(processIncomingMessage);

    // Initialize Queues
    publisherQueue = xQueueCreate(QUEUE_SIZE, sizeof(WateringEvent));
    if (publisherQueue == NULL) {
        printf("error creating the publisherQueue\n");
    }
    wateringQueue = xQueueCreate(QUEUE_SIZE, sizeof(WateringEvent));
    if (wateringQueue == NULL) {
        printf("error creating the wateringQueue\n");
    }

    // Start all tasks (currently using equal priorities)
    xTaskCreate(mqttConnectTask, "MQTTConnectTask", 2048, NULL, 1, &mqttConnectTaskHandle);
    xTaskCreate(mqttLoopTask, "MQTTLoopTask", 4096, NULL, 1, &mqttLoopTaskHandle);
    xTaskCreate(publisherTask, "PublisherTask", 2048, NULL, 1, &publisherTaskHandle);
    xTaskCreate(waterPlantTask, "WaterPlantTask", 2048, NULL, 1, &waterPlantTaskHandle);
    xTaskCreate(waterIntervalTask, "WaterIntervalTask", 2048, NULL, 1, &waterIntervalTaskHandle);
    xTaskCreate(readButtonsTask, "ReadButtonsTask", 2048, NULL, 1, &readButtonsTaskHandle);

    // I tested the stack sizes above by enabling this task which will print
    // the number of words remaining when that task's stack reached its highest
    // xTaskCreate(getStackSizesTask, "getStackSizesTask", 4096, NULL, 1, NULL);
}

void loop() {}

/*
  waterIntervalTask will queue up each plant to be watered fro the configured
  default time. Then it will wait during the configured interval and then loop
*/
void waterIntervalTask(void* parameters) {
    while (true) {
        // Every 24 hours, start watering plant 1
        unsigned long currentMillis = millis();
        if (currentMillis - previousMillis >= INTERVAL) {
            previousMillis = currentMillis;
            for (int i = 0; i < NUM_PLANTS; i++) {
                waterPlant(i, DEFAULT_WATER_TIME, "N/A");
            }
        }
        vTaskDelay(INTERVAL / portTICK_PERIOD_MS);
    }
    vTaskDelete(NULL);
}

/*
  waterPlantTask will wait for WateringEvents on a queue and will then open the
  valve for an amount of time. The delay before closing the valve is done with
  xTaskNotifyWait, allowing it to be interrupted with xTaskNotify. After the
  valve is closed, the WateringEvent is pushed to the queue fro publisherTask
  which will record the WateringEvent in InfluxDB via MQTT and Telegraf
*/
void waterPlantTask(void* parameters) {
    WateringEvent we;
    while (true) {
        if (xQueueReceive(wateringQueue, &we, 0)) {
            // First clear the notifications to prevent a bug that would cause
            // watering to be skipped if I run xTaskNotify when not waiting
            ulTaskNotifyTake(NULL, 0);

            if (we.duration == 0) {
                we.duration = DEFAULT_WATER_TIME;
            }

            // Only water if this valve isn't setup to be skipped
            if (!skipValve[we.plant_position]) {
                unsigned long start = millis();
                plantOn(we.plant_position);
                // Delay for specified watering time with option to interrupt
                xTaskNotifyWait(0x00, ULONG_MAX, NULL, we.duration / portTICK_PERIOD_MS);
                unsigned long stop = millis();
                plantOff(we.plant_position);
                we.duration = stop - start;
                xQueueSend(publisherQueue, &we, portMAX_DELAY);
            } else {
                printf("skipping watering for valve %d\n", we.plant_position);
                skipValve[we.plant_position] = false;
            }
        }
        vTaskDelay(5 / portTICK_PERIOD_MS);
    }
    vTaskDelete(NULL);
}

/*
  plantOn will turn on the correct valve and pump for a specific plant
*/
void plantOn(int id) {
    printf("turning on plant %d\n", id);
    gpio_set_level(plants[id][0], 1);
    gpio_set_level(plants[id][1], 1);
}

/*
  plantOff will turn off the correct valve and pump for a specific plant
*/
void plantOff(int id) {
    printf("turning off plant %d\n", id);
    gpio_set_level(plants[id][0], 0);
    gpio_set_level(plants[id][1], 0);
}

/*
  readButtonsTask will check if any buttons are being pressed
*/
void readButtonsTask(void* parameters) {
    while (true) {
        // Check if any valves need to be stopped and check all buttons
        for (int i = 0; i < NUM_PLANTS; i++) {
            readButton(i);
        }
        readStopButton();
        vTaskDelay(5 / portTICK_PERIOD_MS);
    }
    vTaskDelete(NULL);
}

/*
  publisherTask reads from a queue and publish WateringEvents as an InfluxDB
  line protocol message to MQTT
*/
void publisherTask(void* parameters) {
    WateringEvent we;
    while (true) {
        if (xQueueReceive(publisherQueue, &we, portMAX_DELAY)) {
            char message[50];
            sprintf(message, "water,plant=%d millis=%lu", we.plant_position, we.duration);
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
  mqttConnectTask will periodically attempt to reconnect to MQTT if needed
*/
void mqttConnectTask(void* parameters) {
    while (true) {
        // Connect to MQTT server if not connected already
        if (!client.connected()) {
            printf("attempting MQTT connection...");
            if (client.connect(MQTT_CLIENT_NAME)) {
                printf("connected\n");
                client.subscribe(waterCommandTopic);
                client.subscribe(stopCommandTopic);
                client.subscribe(stopAllCommandTopic);
                client.subscribe(skipCommandTopic);
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
  getStackSizesTask is a tool used for debugging and testing that will give me
  information about the remaining words in each task's stack at its highest
*/
void getStackSizesTask(void* parameters) {
    while (true) {
        printf("mqttConnectTask stack high water mark: %d\n", uxTaskGetStackHighWaterMark(mqttConnectTaskHandle));
        printf("mqttLoopTask stack high water mark: %d\n", uxTaskGetStackHighWaterMark(mqttLoopTaskHandle));
        printf("publisherTask stack high water mark: %d\n", uxTaskGetStackHighWaterMark(publisherTaskHandle));
        printf("waterPlantTask stack high water mark: %d\n", uxTaskGetStackHighWaterMark(waterPlantTaskHandle));
        printf("waterIntervalTask stack high water mark: %d\n", uxTaskGetStackHighWaterMark(waterIntervalTaskHandle));
        printf("readButtonsTask stack high water mark: %d\n", uxTaskGetStackHighWaterMark(readButtonsTaskHandle));
        vTaskDelay(10000 / portTICK_PERIOD_MS);
    }
}

/*
  stopWatering will interrupt the WaterPlantTask. If another plant is in the queue,
  it will begin watering
*/
void stopWatering() {
    xTaskNotify(waterPlantTaskHandle, 0, eNoAction);
}

/*
  stopAllWatering will interrupt the WaterPlantTask and clear the remaining queue
*/
void stopAllWatering() {
    xQueueReset(wateringQueue);
    xTaskNotify(waterPlantTaskHandle, 0, eNoAction);
}

/*
  readButton takes an ID that represents the array index for the valve and button arrays
  and checks if the button is pressed. If the button is pressed, the following is done:
    - stop watering all plants
    - disable watering cycle
    - turn on the valve corresponding to this button
*/
void readButton(int valveID) {
    // Exit if valveID is out of bounds
    if (valveID >= NUM_PLANTS || valveID < 0) {
        return;
    }
    int reading = gpio_get_level(plants[valveID][2]);
    // If the switch changed, due to noise or pressing, reset debounce timer
    if (reading != lastButtonStates[valveID]) {
        lastDebounceTime = millis();
    }

    // Current reading has been the same longer than our delay, so now we can do something
    if ((millis() - lastDebounceTime) > DEBOUNCE_DELAY) {
        // If the button state has changed
        if (reading != buttonStates[valveID]) {
            buttonStates[valveID] = reading;

            // If our button state is HIGH, stop watering others and water this plant
            if (buttonStates[valveID] == HIGH) {
                if (reading == HIGH) {
                    printf("button pressed: %d\n", valveID);
                    waterPlant(valveID, DEFAULT_WATER_TIME, "N/A");
                }
            }
        }
    }
    lastButtonStates[valveID] = reading;
}

/*
  readStopButton is similar to the readButton function, but had to be separated because this
  button does not correspond to a Valve and could not be included in the array of buttons.
*/
void readStopButton() {
    int reading = gpio_get_level(STOP_BUTTON_PIN);
    // If the switch changed, due to noise or pressing, reset debounce timer
    if (reading != lastStopButtonState) {
        lastStopDebounceTime = millis();
    }

    // Current reading has been the same longer than our delay, so now we can do something
    if ((millis() - lastStopDebounceTime) > DEBOUNCE_DELAY) {
        // If the button state has changed
        if (reading != stopButtonState) {
            stopButtonState = reading;

            // If our button state is HIGH, do some things
            if (stopButtonState == HIGH) {
                if (reading == HIGH) {
                    printf("stop button pressed\n");
                    stopWatering();
                }
            }
        }
    }
    lastStopButtonState = reading;
}

/*
  processIncomingMessage is a callback function for the MQTT client that will
  react to incoming messages. Currently, the topics are:
    - waterCommandTopic: accepts a WateringEvent JSON to water a plant for
                         specified time
    - stopCommandTopic: ignores message and stops the currently-watering plant
    - stopAllCommandTopic: ignores message, stops the currently-watering plant,
                           and clears the wateringQueue
    - skipCommandTopic:
*/
void processIncomingMessage(char* topic, byte* message, unsigned int length) {
    printf("message received:\n\ttopic=%s\n\tmessage=%s\n", topic, (char*)message);

    StaticJsonDocument<JSON_CAPACITY> doc;
    DeserializationError err = deserializeJson(doc, message);
    if (err) {
        printf("deserialize failed: %s\n", err.c_str());
    }

    WateringEvent we = {
        doc["plant_position"] | -1,
        doc["duration"] | 0,
        doc["id"] | "N/A"
    };

    if (strcmp(topic, waterCommandTopic) == 0) {
        printf("received command to water plant %d (%s) for %lu\n", we.plant_position, we.id, we.duration);
        waterPlant(we.plant_position, we.duration, we.id);
    } else if (strcmp(topic, stopCommandTopic) == 0) {
        printf("received command to stop watering\n");
        stopWatering();
    } else if (strcmp(topic, stopAllCommandTopic) == 0) {
        printf("received command to stop ALL watering\n");
        stopAllWatering();
    } else if (strcmp(topic, skipCommandTopic) == 0) {
        printf("received command to skip next watering for plant %d (%s)\n", we.plant_position, we.id);
        skipPlant(we.plant_position);
    }
}

/*
  waterPlant pushes a WateringEvent to the queue in order to water a single
  plant. First it will make sure the ID is not out of bounds
*/
void waterPlant(int plant_position, unsigned long duration, const char* id) {
    // Exit if valveID is out of bounds
    if (plant_position >= NUM_PLANTS || plant_position < 0) {
        printf("plant_position %d is out of range, aborting request\n", plant_position);
        return;
    }
    printf("pushing WateringEvent to queue: id=%s, position=%d, time=%lu\n", id, plant_position, duration);
    WateringEvent we = { plant_position, duration, id };
    xQueueSend(wateringQueue, &we, portMAX_DELAY);
}

/*
  skipPlant simply sets the value in the skip array after making sure the plant
  position is valid
*/
void skipPlant(int plant_position) {
    // Exit if valveID is out of bounds
    if (plant_position >= NUM_PLANTS || plant_position < 0) {
        printf("plant_position %d is out of range, aborting request\n", plant_position);
        return;
    }
    skipValve[plant_position] = true;
}
