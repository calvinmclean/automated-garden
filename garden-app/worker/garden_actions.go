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
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"github.com/calvinmclean/automated-garden/garden-app/clock"
	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
)

// ExecuteGardenAction will execute a GardenAction
func (w *Worker) ExecuteGardenAction(ctx context.Context, g *pkg.Garden, input *action.GardenAction) error {
	switch {
	case input.Light != nil:
		err := w.ExecuteLightAction(ctx, g, input.Light)
		if err != nil {
			return fmt.Errorf("unable to execute LightAction: %v", err)
		}
	case input.Fan != nil:
		err := w.ExecuteFanAction(ctx, g, input.Fan)
		if err != nil {
			return fmt.Errorf("unable to execute FanAction: %v", err)
		}
	case input.Stop != nil:
		err := w.ExecuteStopAction(ctx, g, input.Stop)
		if err != nil {
			return fmt.Errorf("unable to execute StopAction: %v", err)
		}
	case input.Update != nil:
		err := w.ExecuteUpdateAction(ctx, g, input.Update)
		if err != nil {
			return fmt.Errorf("unable to execute UpdateAction: %v", err)
		}
	case input.ControllerSetup != nil:
		err := w.ExecuteControllerSetupAction(ctx, g, input.ControllerSetup)
		if err != nil {
			return fmt.Errorf("unable to execute ControllerSetupAction: %v", err)
		}
	case input.FirmwareUpdate != nil:
		err := w.ExecuteFirmwareUpdateAction(ctx, g, input.FirmwareUpdate)
		if err != nil {
			return fmt.Errorf("unable to execute FirmwareUpdateAction: %v", err)
		}
	}
	return nil
}

// ExecuteStopAction sends the message over MQTT to the embedded garden controller
func (w *Worker) ExecuteStopAction(ctx context.Context, g *pkg.Garden, input *action.StopAction) error {
	topicFunc := mqtt.StopTopic
	if input.All {
		topicFunc = mqtt.StopAllTopic
	}
	topic, err := topicFunc(g.TopicPrefix)
	if err != nil {
		return fmt.Errorf("unable to fill MQTT topic template: %v", err)
	}

	return w.mqttClient.Publish(ctx, topic, []byte("no message"))
}

// ExecuteLightAction sends an MQTT message to the garden controller to change the state of the light
func (w *Worker) ExecuteLightAction(ctx context.Context, g *pkg.Garden, input *action.LightAction) error {
	msg, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("unable to marshal LightAction to JSON: %v", err)
	}

	topic, err := mqtt.LightTopic(g.TopicPrefix)
	if err != nil {
		return fmt.Errorf("unable to fill MQTT topic template: %v", err)
	}

	err = w.mqttClient.Publish(ctx, topic, msg)
	if err != nil {
		return fmt.Errorf("unable to publish LightAction: %v", err)
	}

	return nil
}

// ExecuteFanAction sends an MQTT message to the garden controller to turn on the fan for a duration
func (w *Worker) ExecuteFanAction(ctx context.Context, g *pkg.Garden, input *action.FanAction) error {
	msg, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("unable to marshal FanAction to JSON: %v", err)
	}

	topic, err := mqtt.FanTopic(g.TopicPrefix)
	if err != nil {
		return fmt.Errorf("unable to fill MQTT topic template: %v", err)
	}

	err = w.mqttClient.Publish(ctx, topic, msg)
	if err != nil {
		return fmt.Errorf("unable to publish FanAction: %v", err)
	}

	return nil
}

// ExecuteUpdateAction sends an MQTT message to the garden controller with the current configuration
func (w *Worker) ExecuteUpdateAction(ctx context.Context, g *pkg.Garden, input *action.UpdateAction) error {
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

	err = w.mqttClient.Publish(ctx, topic, msg)
	if err != nil {
		return fmt.Errorf("unable to publish UpdateAction: %v", err)
	}

	return nil
}

// sendControllerRequest sends an HTTP POST to the controller's .local hostname first.
// If that fails at the network layer and a stored IP address is available, it retries
// against the IP address before giving up.
func (w *Worker) sendControllerRequest(ctx context.Context, g *pkg.Garden, localURL, path string, body []byte, contentType string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, localURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := w.httpClient.Do(req)
	if err == nil {
		return resp, nil
	}

	if g == nil || g.ControllerInfo == nil || g.ControllerInfo.IPAddress == "" {
		return nil, err
	}

	ipHost := g.ControllerInfo.IPAddress
	if _, _, splitErr := net.SplitHostPort(ipHost); splitErr != nil {
		ipHost = net.JoinHostPort(ipHost, "80")
	}
	ipURL := fmt.Sprintf("http://%s%s", ipHost, path)

	w.logger.Debug("controller request failed, retrying with stored IP address", "local_url", localURL, "ip_url", ipURL, "error", err)

	req, err = http.NewRequestWithContext(ctx, http.MethodPost, ipURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, ipErr := w.httpClient.Do(req)
	if ipErr != nil {
		return nil, fmt.Errorf("unable to send request to %q and fallback to %q: %v, %v", localURL, ipURL, err, ipErr)
	}
	return resp, nil
}

// ExecuteControllerSetupAction sends MQTT connection details to the controller's
// WiFiManager paramsave endpoint
func (w *Worker) ExecuteControllerSetupAction(ctx context.Context, g *pkg.Garden, input *action.ControllerSetupAction) error {
	if input.Server == "" {
		return errors.New("controller_setup action must have server")
	}
	if input.TopicPrefix == "" {
		return errors.New("controller_setup action must have topic_prefix")
	}
	if input.Port <= 0 {
		return errors.New("controller_setup action must have a positive port")
	}

	form := url.Values{}
	form.Set("server", input.Server)
	form.Set("topic_prefix", input.TopicPrefix)
	form.Set("port", strconv.Itoa(input.Port))

	resp, err := w.sendControllerRequest(
		ctx, g,
		w.controllerSetupURLFunc(g.TopicPrefix),
		"/paramsave", []byte(form.Encode()),
		"application/x-www-form-urlencoded",
	)
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
func (w *Worker) ExecuteFirmwareUpdateAction(ctx context.Context, g *pkg.Garden, input *action.FirmwareUpdateAction) error {
	var firmware []byte
	var err error

	if input.Latest {
		firmware, err = w.downloadLatestFirmware(ctx)
		if err != nil {
			return fmt.Errorf("unable to download latest firmware: %v", err)
		}
	} else {
		if len(input.FileData) == 0 {
			return errors.New("firmware_update action must have a file or latest=true")
		}
		firmware = input.FileData

		if len(firmware) > maxFirmwareSize {
			return fmt.Errorf("firmware exceeds maximum size of %d bytes", maxFirmwareSize)
		}
	}

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

	resp, err := w.sendControllerRequest(
		ctx, g,
		w.firmwareUpdateUploadURLFunc(g.TopicPrefix),
		"/u", body.Bytes(),
		writer.FormDataContentType(),
	)
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

func (w *Worker) downloadLatestFirmware(ctx context.Context) ([]byte, error) {
	w.firmwareMutex.Lock()
	defer w.firmwareMutex.Unlock()

	if info, err := os.Stat(w.firmwareFile); err == nil && clock.Since(info.ModTime()) < firmwareCacheTTL {
		cachedData, err := os.ReadFile(w.firmwareFile)
		if err == nil {
			return cachedData, nil
		}
	}

	releaseURL := w.firmwareUpdateReleaseURLFunc()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releaseURL, nil)
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

	req, err = http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
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

	firmware, err := io.ReadAll(io.LimitReader(resp.Body, int64(maxFirmwareSize)+1))
	if err != nil {
		return nil, fmt.Errorf("unable to read asset: %v", err)
	}

	if len(firmware) > maxFirmwareSize {
		return nil, fmt.Errorf("firmware exceeds maximum size of %d bytes", maxFirmwareSize)
	}

	if err := os.MkdirAll(filepath.Dir(w.firmwareFile), 0o750); err != nil {
		return nil, fmt.Errorf("unable to create cache directory: %v", err)
	}
	if err := os.WriteFile(w.firmwareFile, firmware, 0o600); err != nil {
		return nil, fmt.Errorf("unable to write firmware file: %v", err)
	}
	if err := os.Chtimes(w.firmwareFile, clock.Now(), clock.Now()); err != nil {
		return nil, fmt.Errorf("unable to set firmware file modification time: %v", err)
	}

	if w.firmwareTimer == nil {
		w.firmwareTimer = clock.AfterFunc(firmwareCacheTTL, w.deleteFirmwareFile)
	} else {
		w.firmwareTimer.Reset(firmwareCacheTTL)
	}

	return firmware, nil
}
