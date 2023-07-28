#ifndef moisture_h
#define moisture_h

extern TaskHandle_t moistureSensorTaskHandle;

void setupMoistureSensors();
int readMoisturePercentage(int position);
void moistureSensorTask(void* parameters);

#endif
