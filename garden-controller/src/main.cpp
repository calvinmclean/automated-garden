#include <Arduino.h>
#include <stdio.h>
#include <esp_system.h>
#include "driver/gpio.h"
#include "driver/ledc.h"

/* include other files for this program */
#include "config.h"
#include "mqtt.h"
#include "main.h"
#include "wifi_manager.h"
#include "sensors.h"

Config config;

/* FreeRTOS Queue and Task handlers */
QueueHandle_t waterQueue;
TaskHandle_t waterZoneTaskHandle;
QueueHandle_t fanQueue;
TaskHandle_t fanTaskHandle;
QueueHandle_t rebootQueue;
TaskHandle_t rebootTaskHandle;

/* state variables */
int light_state;
int fan_power;

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

void setupFan() {
    fan_power = 0;

    ledc_timer_config_t ledc_timer;
    ledc_timer.speed_mode = LEDC_LOW_SPEED_MODE;
    ledc_timer.duty_resolution = LEDC_TIMER_8_BIT;
    ledc_timer.timer_num = LEDC_TIMER_1;
    ledc_timer.freq_hz = 25000;
    ledc_timer.clk_cfg = LEDC_AUTO_CLK;
    ledc_timer_config(&ledc_timer);

    ledc_channel_config_t ledc_channel;
    ledc_channel.gpio_num = config.fanPin;
    ledc_channel.speed_mode = LEDC_LOW_SPEED_MODE;
    ledc_channel.channel = LEDC_CHANNEL_1;
    ledc_channel.intr_type = LEDC_INTR_DISABLE;
    ledc_channel.timer_sel = LEDC_TIMER_1;
    ledc_channel.duty = 0;
    ledc_channel.hpoint = 0;
    ledc_channel_config(&ledc_channel);

    // Explicitly set fan to 0% on startup so it does not retain a previous duty cycle
    ledc_set_duty(LEDC_LOW_SPEED_MODE, LEDC_CHANNEL_1, 0);
    ledc_update_duty(LEDC_LOW_SPEED_MODE, LEDC_CHANNEL_1);
}

/*
  waterZoneTask will wait for WaterMessages on a queue and will then open the
  valve for an amount of time. The delay before closing the valve is done with
  xTaskNotifyWait, allowing it to be interrupted with xTaskNotify. After the
  valve is closed, the WaterStatusEvent is pushed to the queue for publisherTask
  which will record the WaterStatusEvent in InfluxDB via MQTT and Telegraf
*/
void waterZoneTask(void* parameters) {
  WaterMessage we;
  while (true) {
    if (xQueueReceive(waterQueue, &we, portMAX_DELAY)) {
      // Make copies for START event (publisher frees copies)
      char* zone_id_start = strdup(we.zone_id);
      char* event_id_start = strdup(we.id);
      WaterStatusEvent startEvent = {we.position, 0, zone_id_start, event_id_start, WATER_START};
      xQueueSend(waterPublisherQueue, &startEvent, portMAX_DELAY);

      unsigned long start = millis();
      zoneOn(we.position);
      // Delay for specified watering time with option to interrupt
      BaseType_t notified = xTaskNotifyWait(0x00, ULONG_MAX, NULL, we.duration / portTICK_PERIOD_MS);
      zoneOff(we.position);
      unsigned long elapsed = millis() - start;

      // Use originals for terminal event (publisher frees originals)
      WaterStatus terminalStatus = (notified == pdTRUE) ? WATER_CANCELLED : WATER_COMPLETE;
      WaterStatusEvent terminalEvent = {we.position, elapsed, we.zone_id, we.id, terminalStatus};
      xQueueSend(waterPublisherQueue, &terminalEvent, portMAX_DELAY);
    }
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
  stopAllWatering will interrupt the WaterZoneTask and clear the remaining queue.
  Queued events are sent to the publisher as CANCELLED so the backend knows
  they were discarded.
*/
void stopAllWatering() {
  WaterMessage we;
  while (xQueueReceive(waterQueue, &we, 0)) {
    WaterStatusEvent cancelEvent = {we.position, 0, we.zone_id, we.id, WATER_CANCELLED};
    xQueueSend(waterPublisherQueue, &cancelEvent, portMAX_DELAY);
    // Ownership of we.zone_id and we.id transferred to publisher — DO NOT free
  }
  xTaskNotify(waterZoneTaskHandle, 0, eNoAction);
}

/*
  waterZone pushes a WaterMessage to the queue in order to water a single
  zone. First it will make sure the ID is not out of bounds
*/
void waterZone(WaterMessage we) {
  // Exit if valveID is out of bounds
  if (we.position >= config.numZones || we.position < 0) {
    printf("position %d is out of range, aborting request\n", we.position);
    return;
  }
  printf("pushing WaterMessage to queue: zone_id=%s, position=%d, time=%lu\n", we.zone_id, we.position, we.duration);
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

/*
  fanTask will wait for FanEvents on a queue and will then set the fan PWM
  duty cycle for the specified duration.
*/
void fanTask(void* parameters) {
  FanEvent fe;
  while (true) {
    if (xQueueReceive(fanQueue, &fe, portMAX_DELAY)) {
      printf("running fan at power %d for %lu ms\n", fe.power, fe.duration);
      fan_power = fe.power;
      ledc_set_duty(LEDC_LOW_SPEED_MODE, LEDC_CHANNEL_1, fan_power);
      ledc_update_duty(LEDC_LOW_SPEED_MODE, LEDC_CHANNEL_1);

      // Publish start state
      xQueueSend(fanPublisherQueue, &fan_power, portMAX_DELAY);

      // A power of 0 means turn the fan off immediately and ignore duration
      if (fe.power == 0) {
        continue;
      }

      // Delay for specified duration
      xTaskNotifyWait(0x00, ULONG_MAX, NULL, fe.duration / portTICK_PERIOD_MS);

      // Turn off fan
      fan_power = 0;
      ledc_set_duty(LEDC_LOW_SPEED_MODE, LEDC_CHANNEL_1, 0);
      ledc_update_duty(LEDC_LOW_SPEED_MODE, LEDC_CHANNEL_1);

      // Publish off state
      xQueueSend(fanPublisherQueue, &fan_power, portMAX_DELAY);
    }
  }
  vTaskDelete(NULL);
}

/*
  changeFan will push a FanEvent to the queue to run the fan for a duration
*/
void changeFan(FanEvent fe) {
  printf("pushing FanEvent to queue: duration=%lu, power=%d\n", fe.duration, fe.power);
  xQueueSend(fanQueue, &fe, portMAX_DELAY);
}

void reboot(unsigned long duration) {
    xQueueSend(rebootQueue, &duration, portMAX_DELAY);
}

void rebootTask(void* parameters) {
  unsigned long delay;
  while (true) {
    if (xQueueReceive(rebootQueue, &delay, portMAX_DELAY)) {
      xTaskNotifyWait(0x00, ULONG_MAX, NULL, delay / portTICK_PERIOD_MS);
      ESP.restart();
    }
  }
  vTaskDelete(NULL);
}

#ifndef UNIT_TEST
void setup() {
  initFS();
  setupConfigVars();

  setupZones();

  if (config.light) {
    setupLight();
  }

  if (config.fan) {
    setupFan();
  }

  setupWifiManager();
  setupMQTTMutexAndQueue();
  setupSensors();
  setupMQTT();

  waterQueue = xQueueCreate(QUEUE_SIZE, sizeof(WaterMessage));
  if (waterQueue == NULL) {
    printf("error creating the waterQueue\n");
  }

  fanQueue = xQueueCreate(QUEUE_SIZE, sizeof(FanEvent));
  if (fanQueue == NULL) {
    printf("error creating the fanQueue\n");
  }

  rebootQueue = xQueueCreate(1, sizeof(unsigned long));
  if (rebootQueue == NULL) {
    printf("error creating the rebootQueue\n");
  }

  xTaskCreate(waterZoneTask, "WaterZoneTask", 2048, NULL, 1, &waterZoneTaskHandle);
  xTaskCreate(fanTask, "FanTask", 2048, NULL, 1, &fanTaskHandle);
  xTaskCreate(rebootTask, "RebootTask", 2048, NULL, 1, &rebootTaskHandle);
}

void loop() {}
#endif
