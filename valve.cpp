#include "valve.h"

Valve::Valve(int i, int p, int pump_pin) {
    id = i;
    pin = p;
    pump = pump_pin;
    pinMode(pin, OUTPUT);
    pinMode(pump, OUTPUT);
    off();
}

void Valve::on() {
    Serial.print("turning on valve ");
    Serial.println(id);
    state = HIGH;
    digitalWrite(pin, state);
    digitalWrite(pump, state);
    startMillis = millis();
}

void Valve::off() {
    Serial.print("turning off valve ");
    Serial.println(id);
    state = LOW;
    digitalWrite(pin, state);
    digitalWrite(pump, state);
    startMillis = 0;
}

// If the valve has been open for the specified time, close it
void Valve::offAfterTime(unsigned long time) {
    if (
        state == HIGH &&
        millis() - startMillis >= time
    ) {
        Serial.print("turning off valve ");
        Serial.print(id);
        Serial.println(" because watering time elapsed");
        off();
    }
}