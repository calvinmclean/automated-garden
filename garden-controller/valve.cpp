#include "valve.h"

Valve::Valve(int i, int p, int pump_pin) {
    id = i;
    pin = p;
    pump = pump_pin;
    pinMode(pin, OUTPUT);
    pinMode(pump, OUTPUT);
    off();
    wateringTime = DEFAULT_WATER_TIME;
}

void Valve::on() {
    Serial.print("turning on valve ");
    Serial.println(id);
    state = HIGH;
    digitalWrite(pin, state);
    digitalWrite(pump, state);
    startMillis = millis();
}

void Valve::on(unsigned long time) {
    on();
    wateringTime = time;
}

void Valve::off() {
    Serial.print("turning off valve ");
    Serial.println(id);
    state = LOW;
    digitalWrite(pin, state);
    digitalWrite(pump, state);
    startMillis = 0;
    wateringTime = DEFAULT_WATER_TIME;
}

// If the valve has been open for the specified amount of time, close it
void Valve::offAfterTime() {
    if (
        state == HIGH &&
        millis() - startMillis >= wateringTime
    ) {
        Serial.print("turning off valve ");
        Serial.print(id);
        Serial.println(" because watering time elapsed");
        off();
    }
}