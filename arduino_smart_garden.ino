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

    watering = 0;
    valves[0].on();
}

void loop() {
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

    if (watering >= NUM_VALVES) {
        watering = -1;
    } else if (watering > -1 && valves[watering].state == LOW) {
        watering++;
        if (watering < NUM_VALVES) {
            valves[watering].on();
        }
    }
}

// using a button will interrupt the watering cycle if it is already in progress,
// turning off all valves and setting "watering" to -1, then it starts watering the
// specified plant
void readButton(int valveID) {
    if (digitalRead(buttons[valveID]) == HIGH) {
        stopAllWatering();
        watering = -1;
        valves[valveID].on();
    }
}

void stopAllWatering() {
    for (int i = 0; i < NUM_VALVES; i++) {
        valves[i].off();
    }
}