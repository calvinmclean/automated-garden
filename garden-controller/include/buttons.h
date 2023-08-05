#ifndef buttons_h
#define buttons_h

#include <Arduino.h>
#include "config.h"

#define DEBOUNCE_DELAY 50

void setupButtons();
void readButtonsTask(void* parameters);
void readButton(int valveID);
void readStopButton();

extern TaskHandle_t readButtonsTaskHandle;

#endif
