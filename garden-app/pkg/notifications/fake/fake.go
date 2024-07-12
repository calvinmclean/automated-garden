package fake

import (
	"errors"
	"sync"

	"github.com/mitchellh/mapstructure"
)

type Config struct {
	CreateError      string `mapstructure:"create_error"`
	SendMessageError string `mapstructure:"send_message_error"`
}

type Client struct {
	*Config
}

type Message struct {
	Title   string
	Message string
}

var (
	messages    = []Message{}
	messagesMtx = sync.Mutex{}
)

func NewClient(options map[string]interface{}) (*Client, error) {
	client := &Client{}

	err := mapstructure.Decode(options, &client.Config)
	if err != nil {
		return nil, err
	}

	if client.Config.CreateError != "" {
		return nil, errors.New(client.CreateError)
	}

	return client, nil
}

func (c *Client) SendMessage(title, message string) error {
	if c.SendMessageError != "" {
		return errors.New(c.SendMessageError)
	}
	messagesMtx.Lock()
	messages = append(messages, Message{title, message})
	messagesMtx.Unlock()
	return nil
}

func LastMessage() Message {
	messagesMtx.Lock()
	defer messagesMtx.Unlock()

	if len(messages) == 0 {
		return Message{}
	}
	result := messages[len(messages)-1]
	return result
}

func Messages() []Message {
	messagesMtx.Lock()
	result := make([]Message, len(messages))
	copy(result, messages)
	messagesMtx.Unlock()
	return result
}

func ResetLastMessage() {
	messagesMtx.Lock()
	messages = []Message{}
	messagesMtx.Unlock()
}
