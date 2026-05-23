#include "controller_info.h"
#include <WiFi.h>
#include "version.h"
#include "mqtt.h"

void publishControllerInfo() {
    uint8_t mac[6];
    WiFi.macAddress(mac);
    char macStr[18];
    snprintf(macStr, sizeof(macStr), "%02x:%02x:%02x:%02x:%02x:%02x",
             mac[0], mac[1], mac[2], mac[3], mac[4], mac[5]);

    IPAddress ip = WiFi.localIP();
    char ipStr[16];
    snprintf(ipStr, sizeof(ipStr), "%d.%d.%d.%d", ip[0], ip[1], ip[2], ip[3]);

    char message[150];
    snprintf(message, sizeof(message), "info mac=\"%s\",ip=\"%s\",version=\"%s\"",
             macStr, ipStr, FIRMWARE_VERSION);

    publishInfoMessage(message);
}
