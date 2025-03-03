#ifndef main_h
#define main_h

// Size of FreeRTOS queues
#define QUEUE_SIZE 10

#include "garden_config.h"

struct WaterEvent {
    int position;
    unsigned long duration;
    const char* zone_id;
    const char* id;
    bool done;
};

struct LightEvent {
    const char* state;
};

void waterZone(WaterEvent we);
void zoneOff(int id);
void zoneOn(int id);
void waterZoneTask(void* parameters);
void stopWatering();
void stopAllWatering();
void changeLight(LightEvent le);
void reboot(unsigned long duration);

extern Config config;

#endif
