#include <stdio.h>
#include "driver/gpio.h"

/* include other files for this program */
#include "config.h"
#ifdef ENABLE_WIFI
#include "mqtt.h"
#endif
#ifdef ENABLE_BUTTONS
#include "buttons.h"
#endif

typedef struct WateringEvent {
    int plant_position;
    unsigned long duration;
    const char* id;
};

/* plant/valve variables */
gpio_num_t plants[NUM_PLANTS][3] = PLANTS;

#ifdef ENABLE_WATERING_INTERVAL
/* watering cycle variables */
unsigned long previousMillis = -INTERVAL;
TaskHandle_t waterIntervalTaskHandle;
#endif

/* FreeRTOS Queue and Task handlers */
QueueHandle_t wateringQueue;
TaskHandle_t waterPlantTaskHandle;

void setup() {
    // Prepare pins
    for (int i = 0; i < NUM_PLANTS; i++) {
        // Setup valve pins
        gpio_reset_pin(plants[i][1]);
        gpio_set_direction(plants[i][1], GPIO_MODE_OUTPUT);

        // Setup pump pins
        gpio_reset_pin(plants[i][0]);
        gpio_set_direction(plants[i][0], GPIO_MODE_OUTPUT);
    }

#ifdef ENABLE_WIFI
    setupWifi();
    setupMQTT();
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
    xTaskCreate(waterPlantTask, "WaterPlantTask", 2048, NULL, 1, &waterPlantTaskHandle);
#ifdef ENABLE_WATERING_INTERVAL
    xTaskCreate(waterIntervalTask, "WaterIntervalTask", 2048, NULL, 1, &waterIntervalTaskHandle);
#endif
}

void loop() {}

#ifdef ENABLE_WATERING_INTERVAL
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
#endif

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

            unsigned long start = millis();
            plantOn(we.plant_position);
            // Delay for specified watering time with option to interrupt
            xTaskNotifyWait(0x00, ULONG_MAX, NULL, we.duration / portTICK_PERIOD_MS);
            unsigned long stop = millis();
            plantOff(we.plant_position);
            we.duration = stop - start;
#ifdef ENABLE_WIFI
            xQueueSend(publisherQueue, &we, portMAX_DELAY);
#endif
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
