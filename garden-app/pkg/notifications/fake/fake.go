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
	// lastMessage allows checking the last message that was sent
	lastMessage    = Message{}
	lastMessageMtx = sync.Mutex{}
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
	lastMessageMtx.Lock()
	lastMessage = Message{title, message}
	lastMessageMtx.Unlock()
	return nil
}

func LastMessage() Message {
	lastMessageMtx.Lock()
	result := lastMessage
	lastMessageMtx.Unlock()
	return result
}

func ResetLastMessage() {
	lastMessageMtx.Lock()
	lastMessage = Message{}
	lastMessageMtx.Unlock()
}
