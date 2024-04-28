package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MQTTHandler struct {
	storageClient *storage.Client
	logger        *slog.Logger
}

func NewMQTTHandler(storageClient *storage.Client, logger *slog.Logger) *MQTTHandler {
	return &MQTTHandler{storageClient, logger}
}

func (h *MQTTHandler) getGarden(topicPrefix string) (*pkg.Garden, error) {
	gardens, err := h.storageClient.Gardens.GetAll(context.Background(), nil)
	if err != nil {
		return nil, fmt.Errorf("error getting all gardens: %w", err)
	}
	var garden *pkg.Garden
	for _, g := range gardens {
		if g.TopicPrefix == topicPrefix {
			garden = g
			break
		}
	}
	if garden == nil {
		return nil, errors.New("no garden found")
	}

	return garden, nil
}

func (h *MQTTHandler) getZone(gardenID string, zonePosition int) (*pkg.Zone, error) {
	zones, err := h.storageClient.Zones.GetAll(context.Background(), nil)
	if err != nil {
		return nil, fmt.Errorf("error getting all zones: %w", err)
	}
	var zone *pkg.Zone
	for _, z := range zones {
		if z.GardenID.String() == gardenID &&
			z.Position != nil &&
			int(*z.Position) == zonePosition {
			zone = z
			break
		}
	}
	if zone == nil {
		return nil, errors.New("no zone found")
	}

	return zone, nil
}

func (h *MQTTHandler) Handle(_ mqtt.Client, msg mqtt.Message) {
	logger := h.logger.With("topic", msg.Topic())
	logger.Info("received message", "message", string(msg.Payload()))

	zonePosition, waterDuration, err := parseWaterMessage(msg.Payload())
	if err != nil {
		logger.Error("error parsing message", "error", err)
		return
	}

	topicPrefix := strings.TrimSuffix(msg.Topic(), "/data/water")
	if topicPrefix == "" {
		logger.Error("received message on invalid topic", "error", err)
		return
	}

	garden, err := h.getGarden(topicPrefix)
	if err != nil {
		logger.Error("error getting garden", "topic_prefix", topicPrefix, "error", err)
		return
	}
	logger.Info("found garden with topic-prefix", "topic_prefix", topicPrefix, "garden_id", garden.GetID())

	zone, err := h.getZone(garden.GetID(), zonePosition)
	if err != nil {
		logger.Error("error getting zone", "zone_position", zonePosition, "error", err)
		return
	}
	logger.Info("found zone with position", "zone_position", zonePosition, "zone_id", zone.GetID())

	// TODO: this might end up getting client from garden or zone config instead of using all
	notificationClients, err := h.storageClient.NotificationClientConfigs.GetAll(context.Background(), nil)
	if err != nil {
		logger.Error("error getting all notification clients", "error", err)
		return
	}

	title := fmt.Sprintf("%s finished watering", zone.Name)
	message := fmt.Sprintf("watered for %s", waterDuration.String())

	for _, nc := range notificationClients {
		ncLogger := logger.With(notificationClientIDLogField, nc.GetID())

		err = nc.SendMessage(title, message)
		if err != nil {
			ncLogger.Error("error sending message", "error", err)
			continue
		}

		ncLogger.Info("successfully send notification")
	}
}

func parseWaterMessage(msg []byte) (int, time.Duration, error) {
	p := &parser{msg, 0}
	zonePosition, err := p.readNextInt()
	if err != nil {
		return 0, 0, fmt.Errorf("error parsing zone position: %w", err)
	}

	waterMillis, err := p.readNextInt()
	if err != nil {
		return 0, 0, fmt.Errorf("error parsing watering time: %w", err)
	}
	waterDuration := time.Duration(waterMillis) * time.Millisecond

	return zonePosition, waterDuration, nil
}

type parser struct {
	data []byte
	i    int
}

func (p *parser) readNextInt() (int, error) {
	reading := false
	var n []byte
	for ; p.i < len(p.data); p.i++ {
		c := p.data[p.i]
		if c == ' ' {
			p.i++
			break
		}
		if reading {
			n = append(n, c)
			continue
		}
		if c == '=' {
			reading = true
			continue
		}
	}

	result, err := strconv.Atoi(string(n))
	if err != nil {
		return 0, fmt.Errorf("invalid integer: %w", err)
	}
	return result, nil
}
