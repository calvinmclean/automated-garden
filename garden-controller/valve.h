#ifndef valve_h
#define valve_h

#include "Arduino.h"

#define DEFAULT_WATER_TIME 15000

class Valve {
private:
    int pin;
    int pump;
    unsigned long startMillis;
    unsigned long wateringTime;

public:
    int id;
    int state;
public:
    Valve(int i, int p, int pump_pin);
    void on();
    void on(unsigned long time);
    void off();
    void offAfterTime();
};

#endif