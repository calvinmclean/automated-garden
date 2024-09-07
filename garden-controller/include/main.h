#ifndef main_h
#define main_h

struct WaterEvent {
    int position;
    unsigned long duration;
    const char* id;
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

extern gpio_num_t zones[NUM_ZONES];
extern gpio_num_t pumps[NUM_ZONES];
extern bool lightEnabled;

#endif
