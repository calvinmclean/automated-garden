#include <Arduino.h>
#include <stdio.h>
#include "driver/gpio.h"

/* include other files for this program */
#include "config.h"
#include "mqtt.h"
#include "main.h"
#include "wifi_manager.h"
#include "dht22.h"

Config config;

/* FreeRTOS Queue and Task handlers */
QueueHandle_t waterQueue;
TaskHandle_t waterZoneTaskHandle;
QueueHandle_t rebootQueue;
TaskHandle_t rebootTaskHandle;

/* state variables */
int light_state;

void setupConfigVars() {
    loadConfigFromFile(config);
    printConfig(config);
}

void setupZones() {
    for (int i = 0; i < config.numZones; i++) {
      // Setup valve and pump pins
      gpio_reset_pin(config.valvePins[i]);
      gpio_set_direction(config.valvePins[i], GPIO_MODE_OUTPUT);

      gpio_reset_pin(config.pumpPins[i]);
      gpio_set_direction(config.pumpPins[i], GPIO_MODE_OUTPUT);
    }
}

void setupLight() {
    gpio_reset_pin(config.lightPin);
    gpio_set_direction(config.lightPin, GPIO_MODE_OUTPUT);
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
      // Copy ZoneID and EventID to re-use when sending the completed event
      char* zone_id = strdup(we.zone_id);
      char* event_id = strdup(we.id);

      if (zone_id == nullptr) {
          printf("memory allocation failed for zone_id\n");
          return;
      }
      if (event_id == nullptr) {
          printf("memory allocation failed for event_id\n");
          return;
      }

      free(we.zone_id);
      free(we.id);

      WaterEvent event = {we.position, 0, zone_id, event_id, false};
      // printf("DEBUG: waterZoneTask sends 1: zone_id=%s event_id=%s\n", zone_id, event_id);
      xQueueSend(waterPublisherQueue, &event, portMAX_DELAY);

      unsigned long start = millis();
      zoneOn(we.position);
      // Delay for specified watering time with option to interrupt
      xTaskNotifyWait(0x00, ULONG_MAX, NULL, we.duration / portTICK_PERIOD_MS);
      zoneOff(we.position);
      unsigned long stop = millis();

      event.done = true;
      event.duration = stop - start;
      // printf("DEBUG: waterZoneTask sends 2: zone_id=%s event_id=%s\n", zone_id, event_id);
      xQueueSend(waterPublisherQueue, &event, portMAX_DELAY);

      free(zone_id);
      free(event_id);
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
  if (id < config.numZones) {
    gpio_set_level(config.pumpPins[id], 1);
    gpio_set_level(config.valvePins[id], 1);
  }
}

/*
  zoneOff will turn off the correct valve and pump for a specific zone
*/
void zoneOff(int id) {
  printf("turning off zone %d\n", id);
  if (id < config.numZones) {
    gpio_set_level(config.pumpPins[id], 0);
    gpio_set_level(config.valvePins[id], 0);
  }
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
  if (we.position >= config.numZones || we.position < 0) {
    printf("position %d is out of range, aborting request\n", we.position);
    return;
  }
  printf("pushing WaterEvent to queue: zone_id=%s, position=%d, time=%lu\n", we.zone_id, we.position, we.duration);
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
  gpio_set_level(config.lightPin, light_state);

  // Log data to MQTT if enabled
  xQueueSend(lightPublisherQueue, &light_state, portMAX_DELAY);
}

void reboot(unsigned long duration) {
    xQueueSend(rebootQueue, &duration, portMAX_DELAY);
}

void rebootTask(void* parameters) {
  unsigned long delay;
  while (true) {
    if (xQueueReceive(rebootQueue, &delay, 0)) {
      xTaskNotifyWait(0x00, ULONG_MAX, NULL, delay / portTICK_PERIOD_MS);
      ESP.restart();
    }
    vTaskDelay(5 / portTICK_PERIOD_MS);
  }
  vTaskDelete(NULL);
}

void setup() {
  initFS();
  setupConfigVars();

  setupZones();

  if (config.light) {
    setupLight();
  }

  setupWifiManager();
  setupMQTT();

  if (config.tempHumidity) {
      setupDHT22();
  }

  waterQueue = xQueueCreate(QUEUE_SIZE, sizeof(WaterEvent));
  if (waterQueue == NULL) {
    printf("error creating the waterQueue\n");
  }

  rebootQueue = xQueueCreate(1, sizeof(unsigned long));
  if (rebootQueue == NULL) {
    printf("error creating the rebootQueue\n");
  }

  xTaskCreate(waterZoneTask, "WaterZoneTask", 2048, NULL, 1, &waterZoneTaskHandle);
  xTaskCreate(rebootTask, "RebootTask", 2048, NULL, 1, &rebootTaskHandle);
}

void loop() {}
