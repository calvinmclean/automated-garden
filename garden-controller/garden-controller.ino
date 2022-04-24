#include <stdio.h>
#include "driver/gpio.h"

/* include other files for this program */
#include "config.h"
#include "mqtt.h"
#ifdef ENABLE_BUTTONS
#include "buttons.h"
#endif
#ifdef ENABLE_MOISTURE_SENSORS
#include "moisture.h"
#endif

typedef struct WateringEvent {
    int position;
    unsigned long duration;
    const char* id;
};

typedef struct LightingEvent {
    const char* state;
};

/* zone/valve variables */
gpio_num_t zones[NUM_ZONES][4] = ZONES;

/* FreeRTOS Queue and Task handlers */
QueueHandle_t wateringQueue;
TaskHandle_t waterZoneTaskHandle;

/* state variables */
int light_state;

void setup() {
#ifndef DISABLE_WATERING
    // Prepare pins
    for (int i = 0; i < NUM_ZONES; i++) {
        // Setup valve pins
        gpio_reset_pin(zones[i][1]);
        gpio_set_direction(zones[i][1], GPIO_MODE_OUTPUT);

        // Setup pump pins
        gpio_reset_pin(zones[i][0]);
        gpio_set_direction(zones[i][0], GPIO_MODE_OUTPUT);
    }
#endif

#ifdef LIGHT_PIN
    gpio_reset_pin(LIGHT_PIN);
    gpio_set_direction(LIGHT_PIN, GPIO_MODE_OUTPUT);
    light_state = 0;
#endif

    setupWifi();
    setupMQTT();
#ifdef ENABLE_MOISTURE_SENSORS
    setupMoistureSensors();
#endif

#ifdef ENABLE_BUTTONS
    setupButtons();
#endif

    // Initialize Queues
    wateringQueue = xQueueCreate(QUEUE_SIZE, sizeof(WateringEvent));
    if (wateringQueue == NULL) {
        printf("error creating the wateringQueue\n");
    }

    // Start all tasks (currently using equal priorities)
    xTaskCreate(waterZoneTask, "WaterZoneTask", 2048, NULL, 1, &waterZoneTaskHandle);

#ifdef ENABLE_MQTT_LOGGING
    // Delay 1 second to allow MQTT to connect
    delay(1000);
    if (client.connected()) {
        client.publish(MQTT_LOGGING_TOPIC, "logs message=\"garden-controller setup complete\"");
    } else {
        printf("unable to publish: not connected to MQTT broker\n");
    }
#endif
}

void loop() {}

/*
  waterZoneTask will wait for WateringEvents on a queue and will then open the
  valve for an amount of time. The delay before closing the valve is done with
  xTaskNotifyWait, allowing it to be interrupted with xTaskNotify. After the
  valve is closed, the WateringEvent is pushed to the queue fro publisherTask
  which will record the WateringEvent in InfluxDB via MQTT and Telegraf
*/
void waterZoneTask(void* parameters) {
    WateringEvent we;
    while (true) {
        if (xQueueReceive(wateringQueue, &we, 0)) {
            // First clear the notifications to prevent a bug that would cause
            // watering to be skipped if I run xTaskNotify when not waiting
            ulTaskNotifyTake(NULL, 0);

            if (we.duration == 0) {
                we.duration = DEFAULT_WATER_TIME;
            }

            unsigned long start = millis();
            zoneOn(we.position);
            // Delay for specified watering time with option to interrupt
            xTaskNotifyWait(0x00, ULONG_MAX, NULL, we.duration / portTICK_PERIOD_MS);
            unsigned long stop = millis();
            zoneOff(we.position);
            we.duration = stop - start;
            xQueueSend(waterPublisherQueue, &we, portMAX_DELAY);
        }
        vTaskDelay(5 / portTICK_PERIOD_MS);
    }
    vTaskDelete(NULL);
}

/*
  zoneOn will turn on the correct valve and pump for a specific zone
*/
void zoneOn(int id) {
    printf("turning on zone %d\n", id);
    gpio_set_level(zones[id][0], 1);
    gpio_set_level(zones[id][1], 1);
}

/*
  zoneOff will turn off the correct valve and pump for a specific zone
*/
void zoneOff(int id) {
    printf("turning off zone %d\n", id);
    gpio_set_level(zones[id][0], 0);
    gpio_set_level(zones[id][1], 0);
}

/*
  stopWatering will interrupt the WaterZoneTask. If another zone is in the queue,
  it will begin watering
*/
void stopWatering() {
    xTaskNotify(waterZoneTaskHandle, 0, eNoAction);
}

/*
  stopAllWatering will interrupt the WaterZoneTask and clear the remaining queue
*/
void stopAllWatering() {
    xQueueReset(wateringQueue);
    xTaskNotify(waterZoneTaskHandle, 0, eNoAction);
}

/*
  waterZone pushes a WateringEvent to the queue in order to water a single
  zone. First it will make sure the ID is not out of bounds
*/
void waterZone(WateringEvent we) {
    // Exit if valveID is out of bounds
    if (we.position >= NUM_ZONES || we.position < 0) {
        printf("position %d is out of range, aborting request\n", we.position);
        return;
    }
    printf("pushing WateringEvent to queue: id=%s, position=%d, time=%lu\n", we.id, we.position, we.duration);
    xQueueSend(wateringQueue, &we, portMAX_DELAY);
}

#ifdef LIGHT_PIN
/*
  changeLight will use the state on the LightingEvent to change the state of the light. If the state
  is empty, this will toggle the current state.
  This is a non-blocking operation, so no task or queue is required.
*/
void changeLight(LightingEvent le) {
    if (strlen(le.state) == 0) {
        light_state = !light_state;
    } else if (strcasecmp(le.state, "on") == 0) {
        light_state = 1;
    } else if (strcasecmp(le.state, "off") == 0) {
        light_state = 0;
    } else {
        printf("Unrecognized LightEvent.state, so state will be unchanged\n");
    }
    printf("Setting light state to %d\n", light_state);
    gpio_set_level(LIGHT_PIN, light_state);

    // Log data to MQTT if enabled
    xQueueSend(lightPublisherQueue, &light_state, portMAX_DELAY);
}
#endif
