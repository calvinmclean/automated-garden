package worker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/clock"
	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/notifications"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/notifications/fake"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/calvinmclean/babyapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGardenAction(t *testing.T) {
	garden := &pkg.Garden{
		Name:        "garden",
		TopicPrefix: "garden",
		ControllerConfig: &pkg.ControllerConfig{
			ValvePins: []uint{1, 2, 3},
		},
	}

	tests := []struct {
		name      string
		action    *action.GardenAction
		setupMock func(*mqtt.MockClient, *influxdb.MockClient)
		assert    func(error, *testing.T)
	}{
		{
			"SuccessfulGardenActionWithLightAction",
			&action.GardenAction{
				Light: &action.LightAction{},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient) {
				mqttClient.On("Publish", mock.Anything, "garden/command/light", mock.Anything).Return(nil)
			},
			func(err error, t *testing.T) {
				assert.NoError(t, err)
			},
		},
		{
			"SuccessfulGardenActionWithStopAction",
			&action.GardenAction{
				Stop: &action.StopAction{},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient) {
				mqttClient.On("Publish", mock.Anything, "garden/command/stop", mock.Anything).Return(nil)
			},
			func(err error, t *testing.T) {
				assert.NoError(t, err)
			},
		},
		{
			"SuccessfulGardenActionWithFanAction",
			&action.GardenAction{
				Fan: &action.FanAction{Duration: 1800000, Power: 127},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient) {
				mqttClient.On("Publish", mock.Anything, "garden/command/fan", mock.Anything).Return(nil)
			},
			func(err error, t *testing.T) {
				assert.NoError(t, err)
			},
		},
		{
			"SuccessfulGardenActionWithUpdateAction",
			&action.GardenAction{
				Update: &action.UpdateAction{Config: true},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient) {
				mqttClient.On("Publish", mock.Anything, "garden/command/update_config", mock.Anything).Return(nil)
			},
			func(err error, t *testing.T) {
				assert.NoError(t, err)
			},
		},
		{
			"UpdateActionErrorFalse",
			&action.GardenAction{
				Update: &action.UpdateAction{Config: false},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient) {},
			func(err error, t *testing.T) {
				assert.Error(t, err)
				assert.Equal(t, "unable to execute UpdateAction: update action must have config=true", err.Error())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mqttClient := new(mqtt.MockClient)
			influxdbClient := new(influxdb.MockClient)
			tt.setupMock(mqttClient, influxdbClient)

			err := NewWorker(nil, influxdbClient, mqttClient, slog.Default()).ExecuteGardenAction(context.Background(), garden, tt.action)
			tt.assert(err, t)
			mqttClient.AssertExpectations(t)
			influxdbClient.AssertExpectations(t)
		})
	}
}

func TestLightActionExecute(t *testing.T) {
	now := clock.Now()
	startTime, _ := pkg.StartTimeFromString("23:00:00-07:00")
	garden := &pkg.Garden{
		ID:          babyapi.NewID(),
		Name:        "garden",
		TopicPrefix: "garden",
		LightSchedule: &pkg.LightSchedule{
			Duration:  &pkg.Duration{Duration: 15 * time.Hour},
			StartTime: startTime,
		},
		CreatedAt: &now,
	}

	tests := []struct {
		name      string
		action    *action.LightAction
		setupMock func(*mqtt.MockClient, *influxdb.MockClient)
		assert    func(error, *testing.T)
	}{
		{
			"Successful",
			&action.LightAction{},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient) {
				mqttClient.On("Publish", mock.Anything, "garden/command/light", mock.Anything).Return(nil)
			},
			func(err error, t *testing.T) {
				assert.NoError(t, err)
			},
		},
		{
			"PublishError",
			&action.LightAction{State: pkg.LightStateOff},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient) {
				mqttClient.On("Publish", mock.Anything, "garden/command/light", mock.Anything).Return(errors.New("publish error"))
			},
			func(err error, t *testing.T) {
				if err == nil {
					t.Error("Expected error, but nil was returned")
				}
				if err.Error() != "unable to publish LightAction: publish error" {
					t.Errorf("Unexpected error string: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient, err := storage.NewClient(storage.Config{
				ConnectionString: ":memory:",
			})
			assert.NoError(t, err)
			defer weather.ResetCache()

			mqttClient := new(mqtt.MockClient)
			influxdbClient := new(influxdb.MockClient)
			tt.setupMock(mqttClient, influxdbClient)
			mqttClient.On("Disconnect", uint(100)).Return()
			influxdbClient.On("Close").Return()

			worker := NewWorker(storageClient, influxdbClient, mqttClient, slog.Default())
			err = worker.ScheduleLightActions(garden)
			assert.NoError(t, err)
			worker.StartAsync()

			err = worker.ExecuteLightAction(context.Background(), garden, tt.action)
			tt.assert(err, t)

			worker.Stop()
			mqttClient.AssertExpectations(t)
			influxdbClient.AssertExpectations(t)
		})
	}
}

func TestFanActionExecute(t *testing.T) {
	garden := &pkg.Garden{
		ID:          babyapi.NewID(),
		Name:        "garden",
		TopicPrefix: "garden",
	}

	tests := []struct {
		name      string
		action    *action.FanAction
		setupMock func(*mqtt.MockClient, *influxdb.MockClient)
		assert    func(error, *testing.T)
	}{
		{
			"Successful",
			&action.FanAction{Duration: 1800000, Power: 127},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient) {
				mqttClient.On("Publish", mock.Anything, "garden/command/fan", mock.Anything).Return(nil)
			},
			func(err error, t *testing.T) {
				assert.NoError(t, err)
			},
		},
		{
			"PublishError",
			&action.FanAction{Duration: 1800000, Power: 127},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient) {
				mqttClient.On("Publish", mock.Anything, "garden/command/fan", mock.Anything).Return(errors.New("publish error"))
			},
			func(err error, t *testing.T) {
				if err == nil {
					t.Error("Expected error, but nil was returned")
				}
				if err.Error() != "unable to publish FanAction: publish error" {
					t.Errorf("Unexpected error string: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mqttClient := new(mqtt.MockClient)
			influxdbClient := new(influxdb.MockClient)
			tt.setupMock(mqttClient, influxdbClient)

			err := NewWorker(nil, influxdbClient, mqttClient, slog.Default()).ExecuteFanAction(context.Background(), garden, tt.action)
			tt.assert(err, t)
			mqttClient.AssertExpectations(t)
			influxdbClient.AssertExpectations(t)
		})
	}
}

func TestStopActionExecute(t *testing.T) {
	garden := &pkg.Garden{
		Name:        "garden",
		TopicPrefix: "garden",
	}

	tests := []struct {
		name      string
		action    *action.StopAction
		setupMock func(*mqtt.MockClient, *influxdb.MockClient)
		assert    func(error, *testing.T)
	}{
		{
			"Successful",
			&action.StopAction{},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient) {
				mqttClient.On("Publish", mock.Anything, "garden/command/stop", mock.Anything).Return(nil)
			},
			func(err error, t *testing.T) {
				assert.NoError(t, err)
			},
		},
		{
			"SuccessfulStopAll",
			&action.StopAction{All: true},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient) {
				mqttClient.On("Publish", mock.Anything, "garden/command/stop_all", mock.Anything).Return(nil)
			},
			func(err error, t *testing.T) {
				assert.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mqttClient := new(mqtt.MockClient)
			influxdbClient := new(influxdb.MockClient)
			tt.setupMock(mqttClient, influxdbClient)

			err := NewWorker(nil, influxdbClient, mqttClient, slog.Default()).ExecuteStopAction(context.Background(), garden, tt.action)
			tt.assert(err, t)
			mqttClient.AssertExpectations(t)
			influxdbClient.AssertExpectations(t)
		})
	}
}

func TestControllerSetupActionExecute(t *testing.T) {
	garden := &pkg.Garden{
		Name:        "garden",
		TopicPrefix: "garden",
	}

	tests := []struct {
		name         string
		action       *action.ControllerSetupAction
		serverStatus int
		assert       func(error, *testing.T)
	}{
		{
			"Successful",
			&action.ControllerSetupAction{
				Server:      "192.168.0.1",
				TopicPrefix: "garden",
				Port:        1883,
			},
			http.StatusOK,
			func(err error, t *testing.T) {
				assert.NoError(t, err)
			},
		},
		{
			"SuccessfulRedirect",
			&action.ControllerSetupAction{
				Server:      "192.168.0.1",
				TopicPrefix: "garden",
				Port:        1883,
			},
			http.StatusFound,
			func(err error, t *testing.T) {
				assert.NoError(t, err)
			},
		},
		{
			"MissingServer",
			&action.ControllerSetupAction{
				TopicPrefix: "garden",
				Port:        1883,
			},
			http.StatusOK,
			func(err error, t *testing.T) {
				assert.Error(t, err)
				assert.Equal(t, "controller_setup action must have server", err.Error())
			},
		},
		{
			"MissingTopicPrefix",
			&action.ControllerSetupAction{
				Server: "192.168.0.1",
				Port:   1883,
			},
			http.StatusOK,
			func(err error, t *testing.T) {
				assert.Error(t, err)
				assert.Equal(t, "controller_setup action must have topic_prefix", err.Error())
			},
		},
		{
			"InvalidPort",
			&action.ControllerSetupAction{
				Server:      "192.168.0.1",
				TopicPrefix: "garden",
				Port:        0,
			},
			http.StatusOK,
			func(err error, t *testing.T) {
				assert.Error(t, err)
				assert.Equal(t, "controller_setup action must have a positive port", err.Error())
			},
		},
		{
			"ServerError",
			&action.ControllerSetupAction{
				Server:      "192.168.0.1",
				TopicPrefix: "garden",
				Port:        1883,
			},
			http.StatusInternalServerError,
			func(err error, t *testing.T) {
				assert.Error(t, err)
				assert.Equal(t, "controller setup request returned status 500", err.Error())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			received := url.Values{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/paramsave", r.URL.Path)
				assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

				r.Body = http.MaxBytesReader(w, r.Body, 1024)
				err := r.ParseForm()
				assert.NoError(t, err)
				received = r.PostForm

				w.WriteHeader(tt.serverStatus)
			}))
			defer server.Close()

			worker := NewWorker(nil, nil, nil, slog.Default())
			worker.controllerSetupURLFunc = func(string) string { return server.URL + "/paramsave" }

			err := worker.ExecuteControllerSetupAction(context.Background(), garden, tt.action)
			tt.assert(err, t)

			if err == nil {
				assert.Equal(t, tt.action.Server, received.Get("server"))
				assert.Equal(t, tt.action.TopicPrefix, received.Get("topic_prefix"))
				assert.Equal(t, strconv.Itoa(tt.action.Port), received.Get("port"))
			}
		})
	}
}

func TestFirmwareUpdateActionExecute(t *testing.T) {
	garden := &pkg.Garden{
		Name:        "garden",
		TopicPrefix: "garden",
	}

	firmwareData := []byte("fake firmware binary data")

	tests := []struct {
		name         string
		action       *action.FirmwareUpdateAction
		setupServers func() (releaseURL, uploadURL string)
		assert       func(error, []byte, *testing.T)
	}{
		{
			"SuccessfulDirectUpload",
			&action.FirmwareUpdateAction{
				Latest:   false,
				FileData: firmwareData,
			},
			func() (string, string) {
				return "", newFirmwareUploadServer(t, firmwareData, http.StatusOK)
			},
			func(err error, _ []byte, t *testing.T) {
				assert.NoError(t, err)
			},
		},
		{
			"SuccessfulLatest",
			&action.FirmwareUpdateAction{
				Latest: true,
			},
			func() (string, string) {
				assetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/firmware.bin", r.URL.Path)
					_, _ = w.Write(firmwareData)
				}))

				releaseServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/releases/tags/controller-latest", r.URL.Path)
					_, _ = fmt.Fprintf(w, `{"assets":[{"name":"firmware.bin","browser_download_url":"%s/firmware.bin"}]}`, assetServer.URL)
				}))

				uploadServer := newFirmwareUploadServer(t, firmwareData, http.StatusOK)
				return releaseServer.URL + "/releases/tags/controller-latest", uploadServer
			},
			func(err error, _ []byte, t *testing.T) {
				assert.NoError(t, err)
			},
		},
		{
			"MissingFile",
			&action.FirmwareUpdateAction{
				Latest: false,
			},
			func() (string, string) {
				return "", ""
			},
			func(err error, _ []byte, t *testing.T) {
				assert.Error(t, err)
				assert.Equal(t, "firmware_update action must have a file or latest=true", err.Error())
			},
		},
		{
			"FileTooLarge",
			&action.FirmwareUpdateAction{
				Latest:   false,
				FileData: make([]byte, maxFirmwareSize+1),
			},
			func() (string, string) {
				return "", ""
			},
			func(err error, _ []byte, t *testing.T) {
				assert.Error(t, err)
				assert.Equal(t, fmt.Sprintf("firmware exceeds maximum size of %d bytes", maxFirmwareSize), err.Error())
			},
		},
		{
			"ControllerError",
			&action.FirmwareUpdateAction{
				Latest:   false,
				FileData: firmwareData,
			},
			func() (string, string) {
				return "", newFirmwareUploadServer(t, firmwareData, http.StatusInternalServerError)
			},
			func(err error, _ []byte, t *testing.T) {
				assert.Error(t, err)
				assert.Equal(t, "firmware update request returned status 500", err.Error())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			releaseURL, uploadURL := tt.setupServers()

			worker := NewWorker(nil, nil, nil, slog.Default())
			t.Cleanup(func() {
				_ = os.Remove(worker.firmwareFile)
			})
			if releaseURL != "" {
				worker.firmwareUpdateReleaseURLFunc = func() string { return releaseURL }
			}
			if uploadURL != "" {
				worker.firmwareUpdateUploadURLFunc = func(string) string { return uploadURL }
			}

			err := worker.ExecuteFirmwareUpdateAction(context.Background(), garden, tt.action)
			tt.assert(err, firmwareData, t)
		})
	}
}

func TestFirmwareUpdateActionCachesLatestFirmware(t *testing.T) {
	garden := &pkg.Garden{
		Name:        "garden",
		TopicPrefix: "garden",
	}

	firmwareData := []byte("fake firmware binary data")

	var assetRequests atomic.Int32
	assetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/firmware.bin", r.URL.Path)
		assetRequests.Add(1)
		_, _ = w.Write(firmwareData)
	}))
	defer assetServer.Close()

	releaseServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/releases/tags/controller-latest", r.URL.Path)
		_, _ = fmt.Fprintf(w, `{"assets":[{"name":"firmware.bin","browser_download_url":"%s/firmware.bin"}]}`, assetServer.URL)
	}))
	defer releaseServer.Close()

	uploadServer := newFirmwareUploadServer(t, firmwareData, http.StatusOK)

	worker := NewWorker(nil, nil, nil, slog.Default())
	worker.firmwareUpdateReleaseURLFunc = func() string { return releaseServer.URL + "/releases/tags/controller-latest" }
	worker.firmwareUpdateUploadURLFunc = func(string) string { return uploadServer }

	t.Cleanup(func() {
		_ = os.Remove(worker.firmwareFile)
	})

	for i := 0; i < 2; i++ {
		err := worker.ExecuteFirmwareUpdateAction(context.Background(), garden, &action.FirmwareUpdateAction{
			Latest: true,
		})
		assert.NoError(t, err)
	}

	assert.Equal(t, int32(1), assetRequests.Load(), "asset server should only be hit once")

	cachedData, err := os.ReadFile(worker.firmwareFile)
	assert.NoError(t, err)
	assert.Equal(t, firmwareData, cachedData)
}

func TestFirmwareUpdateActionRefreshesStaleFirmwareCache(t *testing.T) {
	mock := clock.MockTime()
	defer clock.Reset()

	garden := &pkg.Garden{
		Name:        "garden",
		TopicPrefix: "garden",
	}

	firmwareData := []byte("updated firmware binary data")
	staleData := []byte("stale firmware binary data")

	var assetRequests atomic.Int32
	assetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/firmware.bin", r.URL.Path)
		assetRequests.Add(1)
		_, _ = w.Write(firmwareData)
	}))
	defer assetServer.Close()

	releaseServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/releases/tags/controller-latest", r.URL.Path)
		_, _ = fmt.Fprintf(w, `{"assets":[{"name":"firmware.bin","browser_download_url":"%s/firmware.bin"}]}`, assetServer.URL)
	}))
	defer releaseServer.Close()

	uploadServer := newFirmwareUploadServer(t, firmwareData, http.StatusOK)

	worker := NewWorker(nil, nil, nil, slog.Default())
	worker.firmwareUpdateReleaseURLFunc = func() string { return releaseServer.URL + "/releases/tags/controller-latest" }
	worker.firmwareUpdateUploadURLFunc = func(string) string { return uploadServer }

	t.Cleanup(func() {
		_ = os.Remove(worker.firmwareFile)
	})

	// Seed a stale firmware file
	err := os.WriteFile(worker.firmwareFile, staleData, 0o600)
	assert.NoError(t, err)
	staleTime := mock.Now().Add(-11 * time.Minute)
	err = os.Chtimes(worker.firmwareFile, staleTime, staleTime)
	assert.NoError(t, err)

	err = worker.ExecuteFirmwareUpdateAction(context.Background(), garden, &action.FirmwareUpdateAction{
		Latest: true,
	})
	assert.NoError(t, err)

	assert.Equal(t, int32(1), assetRequests.Load(), "asset server should be hit for stale cache")

	cachedData, err := os.ReadFile(worker.firmwareFile)
	assert.NoError(t, err)
	assert.Equal(t, firmwareData, cachedData)
}

func TestFirmwareUpdateActionDeletesCachedFirmwareAfterTTL(t *testing.T) {
	mock := clock.MockTime()
	defer clock.Reset()

	garden := &pkg.Garden{
		Name:        "garden",
		TopicPrefix: "garden",
	}

	firmwareData := []byte("fake firmware binary data")

	assetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/firmware.bin", r.URL.Path)
		_, _ = w.Write(firmwareData)
	}))
	defer assetServer.Close()

	releaseServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/releases/tags/controller-latest", r.URL.Path)
		_, _ = fmt.Fprintf(w, `{"assets":[{"name":"firmware.bin","browser_download_url":"%s/firmware.bin"}]}`, assetServer.URL)
	}))
	defer releaseServer.Close()

	uploadServer := newFirmwareUploadServer(t, firmwareData, http.StatusOK)

	worker := NewWorker(nil, nil, nil, slog.Default())
	worker.firmwareUpdateReleaseURLFunc = func() string { return releaseServer.URL + "/releases/tags/controller-latest" }
	worker.firmwareUpdateUploadURLFunc = func(string) string { return uploadServer }

	t.Cleanup(func() {
		_ = os.Remove(worker.firmwareFile)
	})

	err := worker.ExecuteFirmwareUpdateAction(context.Background(), garden, &action.FirmwareUpdateAction{
		Latest: true,
	})
	assert.NoError(t, err)

	_, err = os.Stat(worker.firmwareFile)
	assert.NoError(t, err)

	mock.Add(firmwareCacheTTL)

	_, err = os.Stat(worker.firmwareFile)
	assert.True(t, os.IsNotExist(err), "cached file should be deleted after TTL")
}

func TestWorkerStopDeletesFirmwareFile(t *testing.T) {
	worker := NewWorker(nil, nil, nil, slog.Default())

	err := os.WriteFile(worker.firmwareFile, []byte("cached firmware"), 0o600)
	assert.NoError(t, err)

	t.Cleanup(func() {
		_ = os.Remove(worker.firmwareFile)
	})

	worker.Stop()

	_, err = os.Stat(worker.firmwareFile)
	assert.True(t, os.IsNotExist(err), "Stop should delete the cached firmware file")
}

func newFirmwareUploadServer(t *testing.T, expectedData []byte, status int) string {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/u", r.URL.Path)
		assert.True(t, strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data"))

		r.Body = http.MaxBytesReader(w, r.Body, int64(maxFirmwareSize+1024))
		err := r.ParseMultipartForm(int64(maxFirmwareSize + 1024))
		assert.NoError(t, err)

		file, header, err := r.FormFile("update")
		assert.NoError(t, err)
		defer func() { _ = file.Close() }()

		assert.Equal(t, "firmware.bin", header.Filename)

		received, err := io.ReadAll(file)
		assert.NoError(t, err)
		assert.Equal(t, expectedData, received)

		w.WriteHeader(status)
	}))
	t.Cleanup(server.Close)

	return server.URL + "/u"
}

func TestControllerSetupActionFallbackToIP(t *testing.T) {
	garden := &pkg.Garden{
		ID:          babyapi.NewID(),
		Name:        "garden",
		TopicPrefix: "garden",
	}

	ipServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/paramsave", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

		r.Body = http.MaxBytesReader(w, r.Body, 1024)
		err := r.ParseForm()
		assert.NoError(t, err)
		assert.Equal(t, "192.168.1.10", r.PostForm.Get("server"))
		assert.Equal(t, "garden", r.PostForm.Get("topic_prefix"))
		assert.Equal(t, "1883", r.PostForm.Get("port"))
		w.WriteHeader(http.StatusOK)
	}))
	defer ipServer.Close()

	garden.ControllerInfo = &pkg.ControllerInfo{IPAddress: ipServer.Listener.Addr().String()}

	localServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("request should not reach local server when fallback succeeds")
	}))
	localServer.Close()

	worker := NewWorker(nil, nil, nil, slog.Default())
	worker.controllerSetupURLFunc = func(string) string { return localServer.URL + "/paramsave" }

	err := worker.ExecuteControllerSetupAction(context.Background(), garden, &action.ControllerSetupAction{
		Server:      "192.168.1.10",
		TopicPrefix: "garden",
		Port:        1883,
	})
	assert.NoError(t, err)
}

func TestControllerSetupActionNoFallbackOnServerError(t *testing.T) {
	garden := &pkg.Garden{
		ID:             babyapi.NewID(),
		Name:           "garden",
		TopicPrefix:    "garden",
		ControllerInfo: &pkg.ControllerInfo{IPAddress: "127.0.0.1:9999"},
	}

	localServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer localServer.Close()

	worker := NewWorker(nil, nil, nil, slog.Default())
	worker.controllerSetupURLFunc = func(string) string { return localServer.URL + "/paramsave" }

	err := worker.ExecuteControllerSetupAction(context.Background(), garden, &action.ControllerSetupAction{
		Server:      "192.168.1.10",
		TopicPrefix: "garden",
		Port:        1883,
	})
	assert.Error(t, err)
	assert.Equal(t, "controller setup request returned status 500", err.Error())
}

func TestFirmwareUpdateActionFallbackToIP(t *testing.T) {
	firmwareData := []byte("fake firmware binary data")

	ipServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/u", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.True(t, strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data"))

		r.Body = http.MaxBytesReader(w, r.Body, int64(maxFirmwareSize+1024))
		err := r.ParseMultipartForm(int64(maxFirmwareSize + 1024))
		assert.NoError(t, err)

		file, header, err := r.FormFile("update")
		assert.NoError(t, err)
		defer func() { _ = file.Close() }()

		assert.Equal(t, "firmware.bin", header.Filename)

		received, err := io.ReadAll(file)
		assert.NoError(t, err)
		assert.Equal(t, firmwareData, received)

		w.WriteHeader(http.StatusOK)
	}))
	defer ipServer.Close()

	garden := &pkg.Garden{
		ID:             babyapi.NewID(),
		Name:           "garden",
		TopicPrefix:    "garden",
		ControllerInfo: &pkg.ControllerInfo{IPAddress: ipServer.Listener.Addr().String()},
	}

	localServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("request should not reach local server when fallback succeeds")
	}))
	localServer.Close()

	worker := NewWorker(nil, nil, nil, slog.Default())
	worker.firmwareUpdateUploadURLFunc = func(string) string { return localServer.URL + "/u" }

	err := worker.ExecuteFirmwareUpdateAction(context.Background(), garden, &action.FirmwareUpdateAction{
		Latest:   false,
		FileData: firmwareData,
	})
	assert.NoError(t, err)
}

func TestFirmwareUpdateActionNoFallbackOnServerError(t *testing.T) {
	firmwareData := []byte("fake firmware binary data")

	localServer := newFirmwareUploadServer(t, firmwareData, http.StatusInternalServerError)

	garden := &pkg.Garden{
		ID:             babyapi.NewID(),
		Name:           "garden",
		TopicPrefix:    "garden",
		ControllerInfo: &pkg.ControllerInfo{IPAddress: "127.0.0.1:9999"},
	}

	worker := NewWorker(nil, nil, nil, slog.Default())
	worker.firmwareUpdateUploadURLFunc = func(string) string { return localServer }

	err := worker.ExecuteFirmwareUpdateAction(context.Background(), garden, &action.FirmwareUpdateAction{
		Latest:   false,
		FileData: firmwareData,
	})
	assert.Error(t, err)
	assert.Equal(t, "firmware update request returned status 500", err.Error())
}

func TestFirmwareUpdateActionConflict(t *testing.T) {
	firmwareData := []byte("fake firmware binary data")

	block := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-block
		wg.Done()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	garden := &pkg.Garden{
		ID:          babyapi.NewID(),
		Name:        "garden",
		TopicPrefix: "garden",
	}

	worker := NewWorker(nil, nil, nil, slog.Default())
	worker.firmwareUpdateUploadURLFunc = func(string) string { return server.URL + "/u" }

	first := &action.GardenAction{
		FirmwareUpdate: &action.FirmwareUpdateAction{
			Latest:   false,
			FileData: firmwareData,
		},
	}
	err := worker.ExecuteGardenAction(context.Background(), garden, first)
	require.NoError(t, err)

	second := &action.GardenAction{
		FirmwareUpdate: &action.FirmwareUpdateAction{
			Latest:   false,
			FileData: firmwareData,
		},
	}
	err = worker.ExecuteGardenAction(context.Background(), garden, second)
	assert.ErrorIs(t, err, ErrFirmwareUpdateInProgress)

	close(block)
	wg.Wait()
}

func TestFirmwareUpdateActionFailureNotification(t *testing.T) {
	fake.Reset()
	defer fake.Reset()

	firmwareData := []byte("fake firmware binary data")

	storageClient, err := storage.NewClient(storage.Config{
		ConnectionString: ":memory:",
	})
	require.NoError(t, err)

	nc := &notifications.Client{
		ID:   babyapi.NewID(),
		Name: "test",
		URL:  "fake://success",
	}
	err = storageClient.NotificationClientConfigs.Set(context.Background(), nc)
	require.NoError(t, err)

	notificationClientID := nc.GetID()
	garden := &pkg.Garden{
		ID:                   babyapi.NewID(),
		Name:                 "garden",
		TopicPrefix:          "garden",
		NotificationClientID: &notificationClientID,
		NotificationSettings: &pkg.NotificationSettings{
			FirmwareChanged: true,
		},
	}
	err = storageClient.Gardens.Set(context.Background(), garden)
	require.NoError(t, err)

	uploadServer := newFirmwareUploadServer(t, firmwareData, http.StatusInternalServerError)

	worker := NewWorker(storageClient, nil, nil, slog.Default())
	worker.firmwareUpdateUploadURLFunc = func(string) string { return uploadServer }

	err = worker.ExecuteGardenAction(context.Background(), garden, &action.GardenAction{
		FirmwareUpdate: &action.FirmwareUpdateAction{
			Latest:   false,
			FileData: firmwareData,
		},
	})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return fake.LastMessage().Title != ""
	}, 2*time.Second, 50*time.Millisecond)

	last := fake.LastMessage()
	assert.Equal(t, "garden: Firmware Update Failed", last.Title)
	assert.Contains(t, last.Message, "firmware update request returned status 500")
}
