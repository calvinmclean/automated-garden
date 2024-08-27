package worker

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const notificationClientIDLogField = "notification_client_id"

func (w *Worker) getGarden(topicPrefix string) (*pkg.Garden, error) {
	gardens, err := w.storageClient.Gardens.GetAll(context.Background(), nil)
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

func (w *Worker) getZone(gardenID string, zonePosition int) (*pkg.Zone, error) {
	zones, err := w.storageClient.Zones.GetAll(context.Background(), nil)
	if err != nil {
		return nil, fmt.Errorf("error getting all zones: %w", err)
	}
	var zone *pkg.Zone
	for _, z := range zones {
		if z.GardenID.String() == gardenID &&
			z.Position != nil &&
			*z.Position == uint(zonePosition) {
			zone = z
			break
		}
	}
	if zone == nil {
		return nil, errors.New("no zone found")
	}

	return zone, nil
}

func (w *Worker) handleWaterCompleteMessage(_ mqtt.Client, msg mqtt.Message) {
	err := w.doWaterCompleteMessage(msg.Topic(), msg.Payload())
	if err != nil {
		w.logger.With("topic", msg.Topic(), "error", err).Error("error handling message")
	}
}

func (w *Worker) doWaterCompleteMessage(topic string, payload []byte) error {
	logger := w.logger.With("topic", topic)
	logger.Info("received message", "message", string(payload))

	zonePosition, waterDuration, err := parseWaterMessage(payload)
	if err != nil {
		return fmt.Errorf("error parsing message: %w", err)
	}

	topicPrefix := strings.TrimSuffix(topic, "/data/water")
	if topicPrefix == "" {
		return fmt.Errorf("received message on invalid topic: %w", err)
	}
	logger = logger.With("topic_prefix", topicPrefix)

	garden, err := w.getGarden(topicPrefix)
	if err != nil {
		return fmt.Errorf("error getting garden with topic-prefix %q: %w", topicPrefix, err)
	}
	logger = logger.With("garden_id", garden.GetID())
	logger.Info("found garden with topic-prefix")

	if garden.GetNotificationClientID() == "" {
		logger.Info("garden does not have notification client", "garden_id", garden.GetID())
		return nil
	}
	logger = logger.With(notificationClientIDLogField, garden.GetNotificationClientID())

	zone, err := w.getZone(garden.GetID(), zonePosition)
	if err != nil {
		return fmt.Errorf("error getting zone with position %d: %w", zonePosition, err)
	}
	logger.Info("found zone with position", "zone_position", zonePosition, "zone_id", zone.GetID())

	notificationClient, err := w.storageClient.NotificationClientConfigs.Get(context.Background(), garden.GetNotificationClientID())
	if err != nil {
		return fmt.Errorf("error getting all notification clients: %w", err)
	}

	title := fmt.Sprintf("%s finished watering", zone.Name)
	message := fmt.Sprintf("watered for %s", waterDuration.String())

	err = notificationClient.SendMessage(title, message)
	if err != nil {
		logger.Error("error sending message", "error", err)
		return err
	}

	logger.Info("successfully send notification")

	return nil
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
