#include "valve.h"

Valve::Valve(int i, gpio_num_t p, gpio_num_t pump_pin) {
    id = i;
    pin = p;
    pump = pump_pin;

    gpio_reset_pin(pin);
    gpio_set_direction(pin, GPIO_MODE_OUTPUT);

    gpio_reset_pin(pump);
    gpio_set_direction(pump, GPIO_MODE_OUTPUT);

    off();
}

void Valve::on() {
    printf("turning on valve %d\n", id);
    gpio_set_level(pin, 1);
    gpio_set_level(pump, 1);
}

void Valve::off() {
    unsigned long result = 0;
    printf("turning off valve %d\n", id);
    gpio_set_level(pin, 0);
    gpio_set_level(pump, 0);
}
