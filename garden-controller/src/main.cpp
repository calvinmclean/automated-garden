#include <Arduino.h>
#include <stdio.h>
#include "driver/gpio.h"

/* include other files for this program */
#include "config.h"
#include "mqtt.h"
#include "main.h"
#include "wifi_manager.h"
#include "dht22.h"


/* zone valve and pump variables */
gpio_num_t valves[NUM_ZONES] = VALVES;
gpio_num_t pumps[NUM_ZONES] = PUMPS;

/* light variables */
bool lightEnabled = LIGHT_ENABLED;
gpio_num_t lightPin = LIGHT_PIN;

bool dht22Enabled = ENABLE_DHT22;

/* FreeRTOS Queue and Task handlers */
QueueHandle_t waterQueue;
TaskHandle_t waterZoneTaskHandle;

/* state variables */
int light_state;

void setupZones() {
    for (int i = 0; i < NUM_ZONES; i++) {
      // Setup valve and pump pins
      gpio_reset_pin(valves[i]);
      gpio_set_direction(valves[i], GPIO_MODE_OUTPUT);

      gpio_reset_pin(pumps[i]);
      gpio_set_direction(pumps[i], GPIO_MODE_OUTPUT);
    }
}

void setupLight() {
    gpio_reset_pin(lightPin);
    gpio_set_direction(lightPin, GPIO_MODE_OUTPUT);
    light_state = 0;
}

/*
  waterZoneTask will wait for WaterEvents on a queue and will then open the
  valve for an amount of time. The delay before closing the valve is done with
  xTaskNotifyWait, allowing it to be interrupted with xTaskNotify. After the
  valve is closed, the WaterEvent is pushed to the queue fro publisherTask
  which will record the WaterEvent in InfluxDB via MQTT and Telegraf
*/
void waterZoneTask(void* parameters) {
  WaterEvent we;
  while (true) {
    if (xQueueReceive(waterQueue, &we, 0)) {
      // First clear the notifications to prevent a bug that would cause
      // watering to be skipped if I run xTaskNotify when not waiting
      ulTaskNotifyTake(NULL, 0);

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
  gpio_set_level(pumps[id], 1);
  gpio_set_level(valves[id], 1);
}

/*
  zoneOff will turn off the correct valve and pump for a specific zone
*/
void zoneOff(int id) {
  printf("turning off zone %d\n", id);
    gpio_set_level(pumps[id], 0);
    gpio_set_level(valves[id], 0);
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
  xQueueReset(waterQueue);
  xTaskNotify(waterZoneTaskHandle, 0, eNoAction);
}

/*
  waterZone pushes a WaterEvent to the queue in order to water a single
  zone. First it will make sure the ID is not out of bounds
*/
void waterZone(WaterEvent we) {
  // Exit if valveID is out of bounds
  if (we.position >= NUM_ZONES || we.position < 0) {
    printf("position %d is out of range, aborting request\n", we.position);
    return;
  }
  printf("pushing WaterEvent to queue: id=%s, position=%d, time=%lu\n", we.id, we.position, we.duration);
  xQueueSend(waterQueue, &we, portMAX_DELAY);
}

/*
  changeLight will use the state on the LightEvent to change the state of the light. If the state
  is empty, this will toggle the current state.
  This is a non-blocking operation, so no task or queue is required.
*/
void changeLight(LightEvent le) {
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
  gpio_set_level(lightPin, light_state);

  // Log data to MQTT if enabled
  xQueueSend(lightPublisherQueue, &light_state, portMAX_DELAY);
}

void setup() {
  setupZones();
  if (lightEnabled) {
    setupLight();
  }

  setupWifiManager();
  setupMQTT();

  if (dht22Enabled) {
      setupDHT22();
  }

  // Initialize Queues
  waterQueue = xQueueCreate(QUEUE_SIZE, sizeof(WaterEvent));
  if (waterQueue == NULL) {
    printf("error creating the waterQueue\n");
  }

  // Start all tasks (currently using equal priorities)
  xTaskCreate(waterZoneTask, "WaterZoneTask", 2048, NULL, 1, &waterZoneTaskHandle);

  // Delay 1 second to allow MQTT to connect
  delay(1000);
  if (client.connected()) {
    client.publish(MQTT_LOGGING_TOPIC, "logs message=\"garden-controller setup complete\"");
  } else {
    printf("unable to publish: not connected to MQTT broker\n");
  }
}

void loop() {}
