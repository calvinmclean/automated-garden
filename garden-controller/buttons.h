#ifndef buttons_h
#define buttons_h

#define DEBOUNCE_DELAY 50

/* button variables */
unsigned long lastDebounceTime = 0;
int buttonStates[NUM_ZONES];
int lastButtonStates[NUM_ZONES];

/* stop button variables */
unsigned long lastStopDebounceTime = 0;
int stopButtonState = LOW;
int lastStopButtonState;

TaskHandle_t readButtonsTaskHandle;

#endif
