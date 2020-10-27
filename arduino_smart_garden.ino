#include "valve.h"

/* Setup pins. They are different for ESP32 and ESP8266 */
#if defined(ESP8266)
#include "esp8266_pins.h"
#elif defined(ESP32)
#include "esp32_pins.h"
#endif

#define NUM_VALVES 3

#define WATER_TIME 15000
#define INTERVAL 86400000 // 24 hours

#define DEBOUNCE_DELAY 50

Valve valves[NUM_VALVES] = {
    Valve(0, VALVE_1_PIN, PUMP_PIN),
    Valve(1, VALVE_2_PIN, PUMP_PIN),
    Valve(2, VALVE_3_PIN, PUMP_PIN)
};

/* button variables */
unsigned long lastDebounceTime = 0;
int buttons[NUM_VALVES] = {BUTTON_1_PIN, BUTTON_2_PIN, BUTTON_3_PIN};
int buttonStates[NUM_VALVES] = {LOW, LOW, LOW};
int lastButtonStates[NUM_VALVES] = {LOW, LOW, LOW};

/* stop button variables */
int stopButtonPin = STOP_BUTTON_PIN;
unsigned long lastStopDebounceTime = 0;
int stopButtonState = LOW;
int lastStopButtonState;

/* watering cycle variables */
unsigned long previousMillis = 0;
int watering = -1;

void setup() {
    Serial.begin(115200);
    for (int i = 0; i < NUM_VALVES; i++) {
        pinMode(buttons[i], INPUT);
    }

    // Start the watering cycle
    watering = 0;
    valves[0].on();
}

void loop() {
    // Check if any valves need to be stopped and check all buttons
    for (int i = 0; i < NUM_VALVES; i++) {
        valves[i].offAfterTime(WATER_TIME);
        readButton(i);
    }
    readStopButton();

    // Every 24 hours, start watering plant 1
    unsigned long currentMillis = millis();
    if (currentMillis - previousMillis >= INTERVAL) {
        previousMillis = currentMillis;
        watering = 0;
        valves[0].on();
    }

    // Manage the watering cycle by starting next plant or ending cycle
    if (watering >= NUM_VALVES) {
        watering = -1;
    } else if (watering > -1 && valves[watering].state == LOW) {
        watering++;
        if (watering < NUM_VALVES) {
            valves[watering].on();
        }
    }
}

/*
  readButton takes an ID that represents the array index for the valve and button arrays
  and checks if the button is pressed. If the button is pressed, the following is done:
    - stop watering all plants
    - reset `watering` variable to disable cycle
    - turn on the valve corresponding to this button
*/
void readButton(int valveID) {
    int reading = digitalRead(buttons[valveID]);
    // If the switch changed, due to noise or pressing, reset debounce timer
    if (reading != lastButtonStates[valveID]) {
        lastDebounceTime = millis();
    }

    // Current reading has been the same longer than our delay, so now we can do something
    if ((millis() - lastDebounceTime) > DEBOUNCE_DELAY) {
        // If the button state has changed
        if (reading != buttonStates[valveID]) {
             buttonStates[valveID] = reading;

            // If our button state is HIGH, do some things
            if (buttonStates[valveID] == HIGH) {
                if (reading == HIGH) {
                    Serial.print("button pressed: ");
                    Serial.println(valveID);

                    stopAllWatering();
                    watering = -1;
                    valves[valveID].on();
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
    int reading = digitalRead(stopButtonPin);
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
                    Serial.print("stop button pressed");

                    stopAllWatering();
                    watering = -1;
                }
            }
        }
    }
    lastStopButtonState = reading;
}

/*
  stopAllWatering will simply loop through all the vavles to turn them off 
*/
void stopAllWatering() {
    for (int i = 0; i < NUM_VALVES; i++) {
        valves[i].off();
    }
}