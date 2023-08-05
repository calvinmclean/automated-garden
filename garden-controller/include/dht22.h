#ifndef dht22_h
#define dht22_h

extern TaskHandle_t dht22TaskHandle;

void setupDHT22();
void dht22PublishTask(void* parameters);

#endif
