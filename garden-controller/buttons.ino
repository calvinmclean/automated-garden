#ifdef ENABLE_BUTTONS

void setupButtons() {
    // Setup button pins and state
    for (int i = 0; i < NUM_ZONES; i++) {
        gpio_set_direction(zones[i][2], GPIO_MODE_INPUT);
        buttonStates[i] = LOW;
        lastButtonStates[i] = LOW;
    }

    xTaskCreate(readButtonsTask, "ReadButtonsTask", 2048, NULL, 1, &readButtonsTaskHandle);
}

/*
  readButtonsTask will check if any buttons are being pressed
*/
void readButtonsTask(void* parameters) {
    while (true) {
        // Check if any valves need to be stopped and check all buttons
        for (int i = 0; i < NUM_ZONES; i++) {
            readButton(i);
        }
        readStopButton();
        vTaskDelay(5 / portTICK_PERIOD_MS);
    }
    vTaskDelete(NULL);
}

/*
  readButton takes an ID that represents the array index for the valve and
  button arrays and checks if the button is pressed. If the button is pressed,
  a WaterEvent for that zone is added to the queue
*/
void readButton(int valveID) {
    // Exit if valveID is out of bounds
    if (valveID >= NUM_ZONES || valveID < 0) {
        return;
    }
    int reading = gpio_get_level(zones[valveID][2]);
    // If the switch changed, due to noise or pressing, reset debounce timer
    if (reading != lastButtonStates[valveID]) {
        lastDebounceTime = millis();
    }

    // Current reading has been the same longer than our delay, so now we can do something
    if ((millis() - lastDebounceTime) > DEBOUNCE_DELAY) {
        // If the button state has changed
        if (reading != buttonStates[valveID]) {
            buttonStates[valveID] = reading;

            // If our button state is HIGH, water the zone
            if (reading == HIGH && buttonStates[valveID] == HIGH) {
                printf("button pressed: %d\n", valveID);
                WaterEvent we = { valveID, DEFAULT_WATER_TIME, "N/A" };
                waterZone(we);
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
#endif
