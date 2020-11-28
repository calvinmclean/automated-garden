#include "valve.h"

Valve::Valve(int i, int p, int pump_pin) {
    id = i;
    pin = p;
    pump = pump_pin;
    skipNext = false;
    pinMode(pin, OUTPUT);
    pinMode(pump, OUTPUT);
    off();
    wateringTime = DEFAULT_WATER_TIME;
}

void Valve::on(unsigned long time) {
    if (skipNext) {
        Serial.printf("skipping watering for valve %d\n", id);
        skipNext = false;
        return;
    }
    if (time > 0) {
        wateringTime = time;
    }
    Serial.printf("turning on valve %d for %lu ms\n", id, time);
    state = HIGH;
    digitalWrite(pin, state);
    digitalWrite(pump, state);
    startMillis = millis();
}

unsigned long Valve::off() {
    // Quickly exit if the valve isn't even watering
    if (state == LOW) { // TODO: "|| startMillis == 0"?
        return 0;
    }
    unsigned long result = 0;
    Serial.printf("turning off valve %d\n", id);
    state = LOW;
    digitalWrite(pin, state);
    digitalWrite(pump, state);
    result = millis() - startMillis;
    startMillis = 0;
    wateringTime = DEFAULT_WATER_TIME;
    return result;
}

/*
  offAfterTime will stop watering if the valve has been open for the specified
  amount of time. It returns the time that was used to water the plant in order
  to publish and save the value
*/
unsigned long Valve::offAfterTime() {
    if (state == LOW || millis() - startMillis < wateringTime) {
        return 0;
    }
    Serial.printf("watering time (%lu ms) elapsed for valve %d\n", wateringTime, id);
    return off();
}

void Valve::setSkipNext() {
    skipNext = true;
}