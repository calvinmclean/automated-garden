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

void setup() {
    // Prepare pins and serial output
    Serial.begin(115200);
    for (int i = 0; i < NUM_VALVES; i++) {
        pinMode(buttons[i], INPUT);
    }

    // Start the watering cycle
    watering = 0;
    valves[0].on();

    // Connect to WiFi and MQTT
    setup_wifi();
    client.setServer(MQTT_ADDRESS, MQTT_PORT);
    client.setCallback(processIncomingMessage);
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
        valves[i].offAfterTime();
        readButton(i);
    }
    readStopButton();

    // Every 24 hours, start watering plant 1
    unsigned long currentMillis = millis();
    if (currentMillis - previousMillis >= INTERVAL) {
        previousMillis = currentMillis;
        watering = 0;
        valves[0].on();
    }

    // Manage the watering cycle by starting next plant or ending cycle
    if (watering >= NUM_VALVES) {
        watering = -1;
    } else if (watering > -1 && valves[watering].state == LOW) {
        watering++;
        if (watering < NUM_VALVES) {
            valves[watering].on();
        }
    }
}

/*
  waterPlant is used for watering a single plant outside of the watering cycle by:
    - turning off all valves
    - resetting cycle tracker
    - watering the specified plant for the specified amount of time
*/
void waterPlant(int id, long time) {
    // Exit if valveID is out of bounds
    if (id >= NUM_VALVES || id < 0) {
        return;
    }
    stopAllWatering();
    watering = -1;
    if (time > 0) {
        valves[id].on(time);
    } else {
        valves[id].on();
    }
}

/*
  readButton takes an ID that represents the array index for the valve and button arrays
  and checks if the button is pressed. If the button is pressed, the following is done:
    - stop watering all plants
    - reset `watering` variable to disable cycle
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

            // If our button state is HIGH, do some things
            if (buttonStates[valveID] == HIGH) {
                if (reading == HIGH) {
                    Serial.print("button pressed: ");
                    Serial.println(valveID);
                    waterPlant(valveID, -1);
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
                    Serial.print("stop button pressed");

                    stopAllWatering();
                    watering = -1;
                }
            }
        }
    }
    lastStopButtonState = reading;
}

/*
  stopAllWatering will simply loop through all the vavles to turn them off
*/
void stopAllWatering() {
    for (int i = 0; i < NUM_VALVES; i++) {
        valves[i].off();
    }
}

/*
  mqttConnect is used to connect to the MQTT server if not already connected. It uses
  millis to only retry the connection every MQTT_RETRY_DELAY seconds without blocking
*/
void mqttConnect() {
    if (!client.connected() && millis() - lastConnectAttempt >= MQTT_RETRY_DELAY) {
        lastConnectAttempt = millis();
        Serial.print("Attempting MQTT connection...");
        if (client.connect("Garden")) {
            Serial.println("connected");
            client.subscribe("garden/water");
        } else {
            Serial.print("failed, rc=");
            Serial.print(client.state());
            Serial.println(" try again in 5 seconds");
        }
    }
}

/*
  processIncomingMessage is a callback function for the MQTT client that will react
  to incoming messages. Currently, the topics are:
    - "garden/water": accepts a WateringEvent JSON to water a plant for specified time
*/
void processIncomingMessage(char* topic, byte* message, unsigned int length) {
    Serial.print("Message arrived on topic: ");
    Serial.print(topic);
    Serial.print(". Message: ");
    Serial.println((char*)message);

    StaticJsonDocument<CAPACITY> doc;
    DeserializationError err = deserializeJson(doc, message);
    if (err) {
        Serial.print(F("deserialize failed "));
        Serial.println(err.c_str());
    }

    if (String(topic) == "garden/water") {
        WateringEvent we = {
            doc["valve_id"],
            doc["water_time"]
        };

        waterPlant(we.valve_id, we.watering_time);
    } else if (String(topic) == "garden/off") {
        stopAllWatering();
    }
}
