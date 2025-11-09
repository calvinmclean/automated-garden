package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/worker"
	"github.com/rs/xid"

	"github.com/calvinmclean/babyapi"
	babytest "github.com/calvinmclean/babyapi/test"
	"github.com/stretchr/testify/assert"
)

func TestWaterRoutine(t *testing.T) {
	worker.CreateNewID = func() xid.ID { return xid.NilID() }
	defer func() { worker.CreateNewID = xid.New }()

	storageClient, err := storage.NewClient(storage.Config{
		Driver: "hashmap",
	})
	assert.NoError(t, err)

	garden := createExampleGarden()
	err = storageClient.Gardens.Set(context.Background(), garden)
	assert.NoError(t, err)

	zones := []pkg.Zone{
		{
			ID:       babyapi.NewID(),
			GardenID: garden.ID.ID,
			Position: pointer(uint(0)),
		},
		{
			ID:       babyapi.NewID(),
			GardenID: garden.ID.ID,
			Position: pointer(uint(1)),
		},
		{
			ID:       babyapi.NewID(),
			GardenID: garden.ID.ID,
			Position: pointer(uint(2)),
		},
	}

	wr := pkg.WaterRoutine{
		ID: babyapi.NewID(),
	}
	for _, z := range zones {
		wr.Steps = append(wr.Steps, pkg.WaterRoutineStep{
			ZoneID:   z.ID,
			Duration: &pkg.Duration{Duration: 1 * time.Second},
		})

		err = storageClient.Zones.Set(context.Background(), &z)
		assert.NoError(t, err)
	}

	api := NewWaterRoutineAPI()
	mqttClient := new(mqtt.MockClient)
	api.setup(storageClient, worker.NewWorker(storageClient, nil, mqttClient, slog.Default()))

	api.worker.StartAsync()
	defer api.worker.Stop()

	t.Run("CreateWaterRoutine", func(t *testing.T) {
		body, err := json.Marshal(wr)
		assert.NoError(t, err)

		r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("%s/%s", waterRoutineBasePath, wr.GetID()), bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-TZ-Offset", "420")
		w := babytest.TestRequest(t, api.API, r)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("CreateWaterRoutine_ErrorZoneNotExist", func(t *testing.T) {
		body := `{"steps": [{"zone_id": "cqsnecmiuvoqlhrmf2o0", "duration": "10s"}]}`

		r := httptest.NewRequest(http.MethodPost, waterRoutineBasePath, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-TZ-Offset", "420")
		w := babytest.TestRequest(t, api.API, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, `{"status":"Invalid request.","error":"unable to get Zone: resource not found"}
`, w.Body.String())
	})

	t.Run("RunRoutine", func(t *testing.T) {
		mqttClient.On("Publish", "test-garden/command/water", fmt.Appendf(nil, `{"duration":1000,"zone_id":"%s","position":0,"id":"00000000000000000000","source":"water_routine"}`, zones[0].GetID())).Return(nil)
		mqttClient.On("Publish", "test-garden/command/water", fmt.Appendf(nil, `{"duration":1000,"zone_id":"%s","position":1,"id":"00000000000000000000","source":"water_routine"}`, zones[1].GetID())).Return(nil)
		mqttClient.On("Publish", "test-garden/command/water", fmt.Appendf(nil, `{"duration":1000,"zone_id":"%s","position":2,"id":"00000000000000000000","source":"water_routine"}`, zones[2].GetID())).Return(nil)
		mqttClient.On("Disconnect", uint(100)).Return()

		r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("%s/%s/run", waterRoutineBasePath, wr.GetID()), http.NoBody)
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-TZ-Offset", "420")
		w := babytest.TestRequest(t, api.API, r)

		assert.Equal(t, http.StatusAccepted, w.Code)

		api.worker.Stop()
		mqttClient.AssertExpectations(t)
	})
}
