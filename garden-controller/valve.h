#ifndef valve_h
#define valve_h

#include "Arduino.h"

#define DEFAULT_WATER_TIME 15000

class Valve {
private:
    int pin;
    int pump;
    unsigned long startMillis;
    bool skipNext;

public:
    int id;
    int state;
    unsigned long wateringTime;
public:
    Valve(int i, int p, int pump_pin);
    void on(unsigned long time);
    unsigned long off();
    unsigned long offAfterTime();
    void setSkipNext();
};

#endif
