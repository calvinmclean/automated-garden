#ifndef valve_h
#define valve_h

#include "Arduino.h"

class Valve {
    private:
        int pin;
        int pump;
        unsigned long startMillis;

    public:
        int id;
        int state;
public:
    Valve(int i, int p, int pump_pin);
    void on();
    void off();
    void offAfterTime(unsigned long time);
};

#endif