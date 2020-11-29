#ifndef valve_h
#define valve_h

#include "driver/gpio.h"
#include <stdio.h>

#define DEFAULT_WATER_TIME 15000

class Valve {
private:
    gpio_num_t pin;
    gpio_num_t pump;

public:
    int id;
public:
    Valve(int i, gpio_num_t p, gpio_num_t pump_pin);
    void on();
    void off();
};

#endif
