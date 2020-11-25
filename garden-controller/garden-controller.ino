#include <ArduinoJson.h>
#include <PubSubClient.h>

/* include other files for this program */
#include "valve.h"
#include "wifi.h"
#include "config.h"

/* setup different pins for ESP32 or ESP8266 */
#if defined(ESP8266)
#include "esp8266_pins.h"
#elif defined(ESP32)
#include "esp32_pins.h"
#endif

#define CAPACITY JSON_OBJECT_SIZE(3) + 40

#define NUM_VALVES 3

#define INTERVAL 86400000 // 24 hours

#define DEBOUNCE_DELAY 50
#define MQTT_RETRY_DELAY 5000

typedef struct WateringEvent {
    int valve_id;
    long watering_time;
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
String waterCommandTopic = "garden/command/water";
String stopCommandTopic = "garden/command/stop";

void setup() {
    // Prepare pins and serial output
    Serial.begin(115200);
    for (int i = 0; i < NUM_VALVES; i++) {
        pinMode(buttons[i], INPUT);
    }

    // Connect to WiFi and MQTT
    setup_wifi();
    client.setServer(MQTT_ADDRESS, MQTT_PORT);
    client.setCallback(processIncomingMessage);

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
        // TODO: consider running this on a separate core when using ESP32 in case
        //       the publishing is slow/blocking
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

    StaticJsonDocument<CAPACITY> doc;
    DeserializationError err = deserializeJson(doc, message);
    if (err) {
        Serial.printf("deserialize failed: %s\n", err.c_str());
    }

    if (String(topic) == waterCommandTopic) {
        WateringEvent we = {
            doc["valve_id"],
            doc["water_time"]
        };
        Serial.printf("received command to water plant %d for %lu\n", we.valve_id, we.watering_time);
        stopAllWatering();
        waterPlant(we.valve_id, we.watering_time);
    } else if (String(topic) == stopCommandTopic) {
        Serial.println("received command to stop watering");
        stopAllWatering();
    }
}

/*
  publishWaterEvent will send an InfluxDB line protocol to MQTT to insert a data
  point into the database. This is meant to be used after turning a valve off, rather
  than turning it on since we don't know if it would be ended early
*/
void publishWaterEvent(int id, unsigned long time) {
    // Now publish event on MQTT topic
    char message[50];
    sprintf(message, "water,plant=%d millis=%lu", id, time);
    Serial.printf("publishing to MQTT:\n\ttopic=%s\n\tmessage=%s\n", "garden/data/water", message);
    client.publish("garden/data/water", message);
}
