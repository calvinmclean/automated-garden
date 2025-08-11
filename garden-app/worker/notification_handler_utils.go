package worker

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
)

const notificationClientIDLogField = "notification_client_id"

func (w *Worker) sendNotificationForGarden(garden *pkg.Garden, title, message string) error {
	if garden.GetNotificationClientID() == "" {
		return errors.New("garden does not have notification client")
	}

	notificationClient, err := w.storageClient.NotificationClientConfigs.Get(context.Background(), garden.GetNotificationClientID())
	if err != nil {
		return fmt.Errorf("error getting all notification clients: %w", err)
	}

	err = notificationClient.SendMessage(title, message)
	if err != nil {
		return err
	}

	return nil
}

func getTopicPrefix(topic string) (string, error) {
	splitTopic := strings.Split(topic, "/")
	if len(splitTopic) != 3 {
		return "", fmt.Errorf("unexpected short topic: %q", topic)
	}

	topicPrefix := splitTopic[0]
	if topicPrefix == "" {
		return "", errors.New("received message on empty topic")
	}

	return topicPrefix, nil
}

func (w *Worker) getGardenForTopic(topic string) (*pkg.Garden, error) {
	splitTopic := strings.SplitN(topic, "/", 2)
	if len(splitTopic) != 2 {
		return nil, fmt.Errorf("unexpected short topic: %q", topic)
	}

	topicPrefix := splitTopic[0]
	if topicPrefix == "" {
		return nil, errors.New("received message on empty topic")
	}

	topicPrefix, err := getTopicPrefix(topic)
	if err != nil {
		return nil, err
	}

	garden, err := w.getGarden(topicPrefix)
	if err != nil {
		return nil, fmt.Errorf("error getting garden with topic-prefix %q: %w", topicPrefix, err)
	}
	return garden, nil
}

func (w *Worker) getGarden(topicPrefix string) (*pkg.Garden, error) {
	gardens, err := w.storageClient.Gardens.Search(context.Background(), "", nil)
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

type parser struct {
	data []byte
	i    int
}

func (p *parser) readNextPair() (string, error) {
	var n []byte
	for ; p.i < len(p.data); p.i++ {
		c := p.data[p.i]
		if c == ' ' || c == ',' {
			p.i++
			break
		}

		n = append(n, c)
	}

	return string(n), nil
}
