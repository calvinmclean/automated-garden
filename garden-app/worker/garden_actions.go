// Package worker handles scheduled watering, health checks, and notifications
package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
)

// ExecuteGardenAction will execute a GardenAction
func (w *Worker) ExecuteGardenAction(g *pkg.Garden, input *action.GardenAction) error {
	switch {
	case input.Light != nil:
		err := w.ExecuteLightAction(g, input.Light)
		if err != nil {
			return fmt.Errorf("unable to execute LightAction: %v", err)
		}
	case input.Fan != nil:
		err := w.ExecuteFanAction(g, input.Fan)
		if err != nil {
			return fmt.Errorf("unable to execute FanAction: %v", err)
		}
	case input.Stop != nil:
		err := w.ExecuteStopAction(g, input.Stop)
		if err != nil {
			return fmt.Errorf("unable to execute StopAction: %v", err)
		}
	case input.Update != nil:
		err := w.ExecuteUpdateAction(g, input.Update)
		if err != nil {
			return fmt.Errorf("unable to execute UpdateAction: %v", err)
		}
	case input.ControllerSetup != nil:
		err := w.ExecuteControllerSetupAction(g, input.ControllerSetup)
		if err != nil {
			return fmt.Errorf("unable to execute ControllerSetupAction: %v", err)
		}
	case input.FirmwareUpdate != nil:
		err := w.ExecuteFirmwareUpdateAction(g, input.FirmwareUpdate)
		if err != nil {
			return fmt.Errorf("unable to execute FirmwareUpdateAction: %v", err)
		}
	}
	return nil
}

// ExecuteStopAction sends the message over MQTT to the embedded garden controller
func (w *Worker) ExecuteStopAction(g *pkg.Garden, input *action.StopAction) error {
	topicFunc := mqtt.StopTopic
	if input.All {
		topicFunc = mqtt.StopAllTopic
	}
	topic, err := topicFunc(g.TopicPrefix)
	if err != nil {
		return fmt.Errorf("unable to fill MQTT topic template: %v", err)
	}

	return w.mqttClient.Publish(topic, []byte("no message"))
}

// ExecuteLightAction sends an MQTT message to the garden controller to change the state of the light
func (w *Worker) ExecuteLightAction(g *pkg.Garden, input *action.LightAction) error {
	msg, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("unable to marshal LightAction to JSON: %v", err)
	}

	topic, err := mqtt.LightTopic(g.TopicPrefix)
	if err != nil {
		return fmt.Errorf("unable to fill MQTT topic template: %v", err)
	}

	err = w.mqttClient.Publish(topic, msg)
	if err != nil {
		return fmt.Errorf("unable to publish LightAction: %v", err)
	}

	return nil
}

// ExecuteFanAction sends an MQTT message to the garden controller to turn on the fan for a duration
func (w *Worker) ExecuteFanAction(g *pkg.Garden, input *action.FanAction) error {
	msg, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("unable to marshal FanAction to JSON: %v", err)
	}

	topic, err := mqtt.FanTopic(g.TopicPrefix)
	if err != nil {
		return fmt.Errorf("unable to fill MQTT topic template: %v", err)
	}

	err = w.mqttClient.Publish(topic, msg)
	if err != nil {
		return fmt.Errorf("unable to publish FanAction: %v", err)
	}

	return nil
}

// ExecuteUpdateAction sends an MQTT message to the garden controller with the current configuration
func (w *Worker) ExecuteUpdateAction(g *pkg.Garden, input *action.UpdateAction) error {
	if !input.Config {
		return errors.New("update action must have config=true")
	}
	if g.ControllerConfig == nil {
		return errors.New("ControllerConfig is nil")
	}

	msg, err := json.Marshal(g.ControllerConfig.ToMessage())
	if err != nil {
		return fmt.Errorf("unable to marshal ControllerConfig to JSON: %v", err)
	}

	topic, err := mqtt.UpdateTopic(g.TopicPrefix)
	if err != nil {
		return fmt.Errorf("unable to fill MQTT topic template: %v", err)
	}

	err = w.mqttClient.Publish(topic, msg)
	if err != nil {
		return fmt.Errorf("unable to publish UpdateAction: %v", err)
	}

	return nil
}

// ExecuteControllerSetupAction sends MQTT connection details to the controller's
// WiFiManager paramsave endpoint
func (w *Worker) ExecuteControllerSetupAction(g *pkg.Garden, input *action.ControllerSetupAction) error {
	if input.Server == "" {
		return errors.New("controller_setup action must have server")
	}
	if input.TopicPrefix == "" {
		return errors.New("controller_setup action must have topic_prefix")
	}
	if input.Port <= 0 {
		return errors.New("controller_setup action must have a positive port")
	}

	endpoint := w.controllerSetupURLFunc(g.TopicPrefix)

	form := url.Values{}
	form.Set("server", input.Server)
	form.Set("topic_prefix", input.TopicPrefix)
	form.Set("port", strconv.Itoa(input.Port))

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("unable to create controller setup request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("unable to send controller setup request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("controller setup request returned status %d", resp.StatusCode)
	}

	return nil
}

// ExecuteFirmwareUpdateAction sends a firmware image to the controller's WiFiManager
// update endpoint. When input.Latest is true, the image is downloaded from the
// controller-latest GitHub release.
func (w *Worker) ExecuteFirmwareUpdateAction(g *pkg.Garden, input *action.FirmwareUpdateAction) error {
	var firmware []byte
	var err error

	if input.Latest {
		firmware, err = w.downloadLatestFirmware()
		if err != nil {
			return fmt.Errorf("unable to download latest firmware: %v", err)
		}
	} else {
		if len(input.FileData) == 0 {
			return errors.New("firmware_update action must have a file or latest=true")
		}
		firmware = input.FileData
	}

	if len(firmware) > maxFirmwareSize {
		return fmt.Errorf("firmware exceeds maximum size of %d bytes", maxFirmwareSize)
	}

	endpoint := w.firmwareUpdateUploadURLFunc(g.TopicPrefix)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("update", firmwareAssetName)
	if err != nil {
		return fmt.Errorf("unable to create firmware form file: %v", err)
	}
	_, err = part.Write(firmware)
	if err != nil {
		return fmt.Errorf("unable to write firmware form file: %v", err)
	}
	err = writer.Close()
	if err != nil {
		return fmt.Errorf("unable to close firmware multipart writer: %v", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, endpoint, body)
	if err != nil {
		return fmt.Errorf("unable to create firmware update request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("unable to send firmware update request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("firmware update request returned status %d", resp.StatusCode)
	}

	return nil
}

type githubRelease struct {
	Assets []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func (w *Worker) downloadLatestFirmware() ([]byte, error) {
	releaseURL := w.firmwareUpdateReleaseURLFunc()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, releaseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to create release request: %v", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch release: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("release request returned status %d", resp.StatusCode)
	}

	var release githubRelease
	decoder := json.NewDecoder(io.LimitReader(resp.Body, 1024*1024))
	err = decoder.Decode(&release)
	if err != nil {
		return nil, fmt.Errorf("unable to decode release response: %v", err)
	}

	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == firmwareAssetName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}
	if downloadURL == "" {
		return nil, fmt.Errorf("release does not contain %q asset", firmwareAssetName)
	}

	req, err = http.NewRequestWithContext(context.Background(), http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to create asset request: %v", err)
	}

	resp, err = w.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch asset: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("asset request returned status %d", resp.StatusCode)
	}

	return io.ReadAll(io.LimitReader(resp.Body, int64(maxFirmwareSize)+1))
}
