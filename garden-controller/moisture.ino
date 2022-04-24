#ifdef ENABLE_MOISTURE_SENSORS

void setupMoistureSensors() {
    for (int i = 0; i < NUM_ZONES; i++) {
        if (zones[i][3] == GPIO_NUM_MAX) {
            continue;
        }
        gpio_reset_pin(zones[i][3]);
        gpio_set_direction(zones[i][3], GPIO_MODE_INPUT);
    }

    xTaskCreate(moistureSensorTask, "MoistureSensorTask", 2048, NULL, 1, &moistureSensorTaskHandle);
}

int readMoisturePercentage(int position) {
    int value = analogRead(zones[position][3]);
    printf("Moisture value: %d\n", value);
    int percentage = map(value, MOISTURE_SENSOR_AIR_VALUE, MOISTURE_SENSOR_WATER_VALUE, 0, 100);
    printf("Moisture percentage: %d\n", percentage);
    if (percentage < 0) {
        percentage = 0;
    } else if (percentage > 100) {
        percentage = 100;
    }
    return percentage;
}

void moistureSensorTask(void* parameters) {
    while (true) {
        for (int zone = 0; zone < NUM_ZONES; zone++) {
            if (zones[zone][3] == GPIO_NUM_MAX) {
                continue;
            }
            int percentage = readMoisturePercentage(zone);
            char message[50];
            sprintf(message, "moisture,zone=%d value=%d", zone, percentage);
            if (client.connected()) {
                printf("publishing to MQTT:\n\ttopic=%s\n\tmessage=%s\n", moistureDataTopic, message);
                client.publish(moistureDataTopic, message);
            } else {
                printf("unable to publish: not connected to MQTT broker\n");
            }
        }
        vTaskDelay(MOISTURE_SENSOR_INTERVAL / portTICK_PERIOD_MS);
    }
    vTaskDelete(NULL);
}

#endif
