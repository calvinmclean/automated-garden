#include <ArduinoJson.h>
#include <PubSubClient.h>

/* include other files for this program */
#include "valve.h"
#include "wifi.h"
#include "config.h"

#define JSON_CAPACITY JSON_OBJECT_SIZE(3) + 40
#define QUEUE_SIZE 10

#define NUM_VALVES 3

#define INTERVAL 86400000 // 24 hours

#define DEBOUNCE_DELAY 50
#define MQTT_RETRY_DELAY 5000

typedef struct WateringEvent {
    int valve_id;
    unsigned long watering_time;
};

Valve valves[NUM_VALVES] = {
    Valve(0, VALVE_1_PIN, PUMP_PIN),
    Valve(1, VALVE_2_PIN, PUMP_PIN),
    Valve(2, VALVE_3_PIN, PUMP_PIN)
};

/* button variables */
unsigned long lastDebounceTime = 0;
int buttons[NUM_VALVES] = { BUTTON_1_PIN, BUTTON_2_PIN, BUTTON_3_PIN };
int buttonStates[NUM_VALVES] = { LOW, LOW, LOW };
int lastButtonStates[NUM_VALVES] = { LOW, LOW, LOW };

/* stop button variables */
unsigned long lastStopDebounceTime = 0;
int stopButtonState = LOW;
int lastStopButtonState;

/* watering cycle variables */
unsigned long previousMillis = 0;
int watering = -1;

/* MQTT variables */
unsigned long lastConnectAttempt = 0;
PubSubClient client(wifiClient);
const char* waterCommandTopic = "garden/command/water";
const char* stopCommandTopic = "garden/command/stop";
const char* waterDataTopic = "garden/data/water";

/* */
QueueHandle_t queue;

void setup() {
    // Prepare pins and serial output
    Serial.begin(115200);
    Serial.printf("running setup on core %d\n", xPortGetCoreID());
    for (int i = 0; i < NUM_VALVES; i++) {
        pinMode(buttons[i], INPUT);
    }

    // Connect to WiFi and MQTT
    setup_wifi();
    client.setServer(MQTT_ADDRESS, MQTT_PORT);
    client.setCallback(processIncomingMessage);

    // Setup queue and start PublisherTask in second core
    queue = xQueueCreate(QUEUE_SIZE, sizeof(WateringEvent));
    if (queue == NULL) {
        Serial.println("error creating the queue");
    }
    // TODO: Figure out accurate stackSize instead of 10000
    xTaskCreatePinnedToCore(publisherTask, "PublisherTask", 10000, NULL, 1, NULL, 0);

    // Start the watering cycle
    watering = 0;
    waterPlant(0);
}

void loop() {
    // Connect to MQTT server if not connected already
    mqttConnect();
    // Run MQTT loop to process incoming messages if connected
    if (client.connected()) {
        client.loop();
    }

    // Check if any valves need to be stopped and check all buttons
    for (int i = 0; i < NUM_VALVES; i++) {
        unsigned long t = valves[i].offAfterTime();
        if (t > 0) {
            publishWaterEvent(i, t);
        }
        readButton(i);
    }
    readStopButton();

    // Every 24 hours, start watering plant 1
    unsigned long currentMillis = millis();
    if (currentMillis - previousMillis >= INTERVAL) {
        previousMillis = currentMillis;
        watering = 0;
        waterPlant(0);
    }

    // Manage the watering cycle by starting next plant or ending cycle
    if (watering >= NUM_VALVES) {
        stopAllWatering();
    } else if (watering > -1 && valves[watering].state == LOW) {
        watering++;
        if (watering < NUM_VALVES) {
            waterPlant(watering);
        }
    }
}

/*
  waterPlant is used for watering a single plant. Even though this is a super basic
  function, it will make the rest of the code more clear
*/
void waterPlant(int id, long time) {
    // Exit if valveID is out of bounds
    if (id >= NUM_VALVES || id < 0) {
        Serial.printf("valve ID %d is out of range, aborting request\n", id);
        return;
    }
    valves[id].on(time);
}

// Simple helper to use the above function with fewer args
void waterPlant(int id) {
    waterPlant(id, 0);
}

/*
  readButton takes an ID that represents the array index for the valve and button arrays
  and checks if the button is pressed. If the button is pressed, the following is done:
    - stop watering all plants
    - disable watering cycle
    - turn on the valve corresponding to this button
*/
void readButton(int valveID) {
    // Exit if valveID is out of bounds
    if (valveID >= NUM_VALVES || valveID < 0) {
        return;
    }
    int reading = digitalRead(buttons[valveID]);
    // If the switch changed, due to noise or pressing, reset debounce timer
    if (reading != lastButtonStates[valveID]) {
        lastDebounceTime = millis();
    }

    // Current reading has been the same longer than our delay, so now we can do something
    if ((millis() - lastDebounceTime) > DEBOUNCE_DELAY) {
        // If the button state has changed
        if (reading != buttonStates[valveID]) {
            buttonStates[valveID] = reading;

            // If our button state is HIGH, stop watering others and water this plant
            if (buttonStates[valveID] == HIGH) {
                if (reading == HIGH) {
                    Serial.printf("button pressed: %d\n", valveID);
                    stopAllWatering();
                    waterPlant(valveID, 0);
                }
            }
        }
    }
    lastButtonStates[valveID] = reading;
}

/*
  readStopButton is similar to the readButton function, but had to be separated because this
  button does not correspond to a Valve and could not be included in the array of buttons.
*/
void readStopButton() {
    int reading = digitalRead(STOP_BUTTON_PIN);
    // If the switch changed, due to noise or pressing, reset debounce timer
    if (reading != lastStopButtonState) {
        lastStopDebounceTime = millis();
    }

    // Current reading has been the same longer than our delay, so now we can do something
    if ((millis() - lastStopDebounceTime) > DEBOUNCE_DELAY) {
        // If the button state has changed
        if (reading != stopButtonState) {
            stopButtonState = reading;

            // If our button state is HIGH, do some things
            if (stopButtonState == HIGH) {
                if (reading == HIGH) {
                    Serial.println("stop button pressed");
                    stopAllWatering();
                }
            }
        }
    }
    lastStopButtonState = reading;
}

/*
  stopAllWatering will stop watering all plants and disable the cycle
*/
void stopAllWatering() {
    watering = -1;
    for (int i = 0; i < NUM_VALVES; i++) {
        unsigned long t = valves[i].off();
        if (t > 0) {
            publishWaterEvent(i, t);
        }
    }
}

/*
  mqttConnect is used to connect to the MQTT server if not already connected. It uses
  millis to only retry the connection every MQTT_RETRY_DELAY seconds without blocking
*/
void mqttConnect() {
    if (!client.connected() && millis() - lastConnectAttempt >= MQTT_RETRY_DELAY) {
        lastConnectAttempt = millis();
        Serial.print("attempting MQTT connection...");
        if (client.connect("Garden")) {
            Serial.println("connected");
            client.subscribe(waterCommandTopic);
            client.subscribe(stopCommandTopic);
        } else {
            Serial.printf("failed, rc=%zu\n", client.state());
        }
    }
}

/*
  processIncomingMessage is a callback function for the MQTT client that will react
  to incoming messages. Currently, the topics are:
    - waterCommandTopic: accepts a WateringEvent JSON to water a plant for specified time
    - stopCommandTopic: ignores message and stops all watering
*/
void processIncomingMessage(char* topic, byte* message, unsigned int length) {
    Serial.printf("message received:\n\ttopic=%s\n\tmessage=%s\n", topic, (char*)message);

    StaticJsonDocument<JSON_CAPACITY> doc;
    DeserializationError err = deserializeJson(doc, message);
    if (err) {
        Serial.printf("deserialize failed: %s\n", err.c_str());
    }

    if (strcmp(topic, waterCommandTopic) == 0) {
        WateringEvent we = {
            doc["valve_id"],
            doc["water_time"]
        };
        Serial.printf("received command to water plant %d for %lu\n", we.valve_id, we.watering_time);
        stopAllWatering();
        waterPlant(we.valve_id, we.watering_time);
    } else if (strcmp(topic, stopCommandTopic) == 0) {
        Serial.println("received command to stop watering");
        stopAllWatering();
    }
}

/*
  publishWaterEvent will put a WateringEvent on the queue to be picked up by the PublisherTask
  on the second core which will record the watering event in InfluxDB via MQTT and Telegraf
*/
void publishWaterEvent(int id, unsigned long time) {
    WateringEvent we = { id, time };
    xQueueSend(queue, &we, portMAX_DELAY);
}

/*
  publisherTask runs a continuous loop where it will read from a queue and publish WateringEvents
  as an InfluxDB line protocol message to MQTT
*/
void publisherTask(void* parameters) {
    Serial.printf("starting PublisherTask on core %d\n", xPortGetCoreID());
    WateringEvent we;
    while (true) {
        if (xQueueReceive(queue, &we, portMAX_DELAY)) {
            char message[50];
            sprintf(message, "water,plant=%d millis=%lu", we.valve_id, we.watering_time);
            Serial.printf("publishing to MQTT:\n\ttopic=%s\n\tmessage=%s\n", waterDataTopic, message);
            client.publish(waterDataTopic, message);
        }
    }
}
