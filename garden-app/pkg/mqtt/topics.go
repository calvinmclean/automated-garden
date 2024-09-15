package mqtt

import (
	"bytes"
	"html/template"
)

const (
	waterTopicTemplate   = "{{.Garden}}/command/water"
	stopTopicTemplate    = "{{.Garden}}/command/stop"
	stopAllTopicTemplate = "{{.Garden}}/command/stop_all"
	lightTopicTemplate   = "{{.Garden}}/command/light"
	updateTopicTemplate  = "{{.Garden}}/command/update"
)

// WaterTopic returns the topic string for watering a zone
func WaterTopic(topicPrefix string) (string, error) {
	return executeTopicTemplate(waterTopicTemplate, topicPrefix)
}

// StopTopic returns the topic string for stopping watering a single zone
func StopTopic(topicPrefix string) (string, error) {
	return executeTopicTemplate(stopTopicTemplate, topicPrefix)
}

// StopAllTopic returns the topic string for stopping watering all zones in a garden
func StopAllTopic(topicPrefix string) (string, error) {
	return executeTopicTemplate(stopAllTopicTemplate, topicPrefix)
}

// LightTopic returns the topic string for changing the light state in a Garden
func LightTopic(topicPrefix string) (string, error) {
	return executeTopicTemplate(lightTopicTemplate, topicPrefix)
}

// UpdateTopic returns the topic string for updating a controller
func UpdateTopic(topicPrefix string) (string, error) {
	return executeTopicTemplate(updateTopicTemplate, topicPrefix)
}

// executeTopicTemplate is a helper function used by all the exported topic evaluation functions
func executeTopicTemplate(templateString string, topicPrefix string) (string, error) {
	t := template.Must(template.New("topic").Parse(templateString))
	var result bytes.Buffer
	data := map[string]string{"Garden": topicPrefix}
	err := t.Execute(&result, data)
	return result.String(), err
}
