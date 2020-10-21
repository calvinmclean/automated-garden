#include "valve.h"

#define D0 16
#define D1 5
#define D2 4
#define D3 0
#define D6 12
#define D7 13
#define D8 15

#define NUM_VALVES 3

// 3 seconds = approx 32 ml water
// 15 seconds = approx 190 ml water
#define WATER_TIME 15000

#define INTERVAL 86400000 // 24 hours

Valve valves[NUM_VALVES] = {
    Valve(0, D0, D3),
    Valve(1, D1, D3),
    Valve(2, D2, D3)
};

int buttons[NUM_VALVES] = {D6, D7, D8};

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
    if (digitalRead(buttons[valveID]) == HIGH) {
        stopAllWatering();
        watering = -1;
        valves[valveID].on();
    }
}

/*
  stopAllWatering will simply loop through all the vavles to turn them off 
*/
void stopAllWatering() {
    for (int i = 0; i < NUM_VALVES; i++) {
        valves[i].off();
    }
}