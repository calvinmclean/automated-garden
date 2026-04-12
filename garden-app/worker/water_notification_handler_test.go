package worker

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"testing"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/notifications"
	fake_notification "github.com/calvinmclean/automated-garden/garden-app/pkg/notifications/fake"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/babyapi"
	"github.com/rs/xid"
	"github.com/stretchr/testify/require"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/cassette"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/recorder"
)

func urlMethodMatcher(r *http.Request, cassetteReq cassette.Request) bool {
	return r.Method == cassetteReq.Method && r.URL.String() == cassetteReq.URL
}

func TestParseWaterMessage(t *testing.T) {
	tests := []struct {
		in       string
		expected action.WaterMessage
		error    string
	}{
		{
			"water,zone=1,zone_id=\"zoneID\",id=\"eventID\" millis=6000",
			action.WaterMessage{
				Position: 1,
				Duration: 6000,
				ZoneID:   "zoneID",
				EventID:  "eventID",
			},
			"",
		},
		{
			"water,zone=100,zone_id=\"zoneID\",id=\"eventID\" millis=1",
			action.WaterMessage{
				Position: 100,
				Duration: 1,
				ZoneID:   "zoneID",
				EventID:  "eventID",
			},
			"",
		},
		{
			"water,status=complete,zone=0,zone_id=\"zoneID\",id=\"eventID\" millis=0",
			action.WaterMessage{
				Position: 0,
				Duration: 0,
				ZoneID:   "zoneID",
				EventID:  "eventID",
				Start:    false,
			},
			"",
		},
		{
			"water,status=start,zone=0,zone_id=\"zoneID\",id=\"eventID\" millis=0",
			action.WaterMessage{
				Position: 0,
				Duration: 0,
				ZoneID:   "zoneID",
				EventID:  "eventID",
				Start:    true,
			},
			"",
		},
		{
			"water,zone=-1,zone_id=zoneID,id=eventID millis=0",
			action.WaterMessage{},
			`invalid integer for position: strconv.ParseUint: parsing "-1": invalid syntax`,
		},
		{
			"water,zone=0,zone_id=zoneID,id=eventID millis=X",
			action.WaterMessage{},
			"invalid integer for millis: strconv.ParseInt: parsing \"X\": invalid syntax",
		},
		{
			"water,status=X,zone=0,zone_id=zoneID,id=eventID millis=1",
			action.WaterMessage{},
			`invalid status: "X"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			waterMessage, err := parseWaterMessage([]byte(tt.in))
			if tt.error == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Equal(t, tt.error, err.Error())
			}
			require.Equal(t, tt.expected, waterMessage)
		})
	}
}

func TestHandleMessage(t *testing.T) {
	storageClient, err := storage.NewClient(storage.Config{
		ConnectionString: ":memory:",
	})
	require.NoError(t, err)

	handler := NewWorker(storageClient, nil, nil, slog.Default())

	t.Run("ErrorParsingMessage", func(t *testing.T) {
		err = handler.doWaterCompleteMessage("garden/data/water", []byte{})
		require.Error(t, err)
		require.Equal(t, "error getting garden with topic-prefix \"garden\": no garden found", err.Error())
	})

	zoneID := babyapi.NewID()
	t.Run("ErrorGettingGarden", func(t *testing.T) {
		msg := fmt.Appendf(nil, "water,zone=0 millis=6000 zone_id=%s id=eventID", zoneID.String())
		err = handler.doWaterCompleteMessage("garden/data/water", msg)
		require.Error(t, err)
		require.Equal(t, "error getting garden with topic-prefix \"garden\": no garden found", err.Error())
	})

	garden := &pkg.Garden{
		ID:                   babyapi.NewID(),
		TopicPrefix:          "garden",
		Name:                 "garden",
		NotificationClientID: nil,
	}
	err = storageClient.Gardens.Set(context.Background(), garden)
	require.NoError(t, err)

	zero := uint(0)
	zone := &pkg.Zone{
		ID:       zoneID,
		GardenID: garden.ID.ID,
		Position: &zero,
	}
	err = storageClient.Zones.Set(context.Background(), zone)
	require.NoError(t, err)

	t.Run("SuccessfulWithNoNotificationClients", func(t *testing.T) {
		msg := fmt.Appendf(nil, "water,zone=0 millis=6000 zone_id=%s id=eventID", zoneID.String())
		err = handler.doWaterCompleteMessage("garden/data/water", msg)
		require.NoError(t, err)
	})

	t.Run("CreateNotificationClient", func(t *testing.T) {
		nc := &notifications.Client{
			ID:  babyapi.NewID(),
			URL: "pushover://shoutrrr:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa@aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/",
		}

		err := storageClient.NotificationClientConfigs.Set(context.Background(), nc)
		require.NoError(t, err)

		ncID := nc.GetID()
		apiErr := garden.Patch(&pkg.Garden{NotificationClientID: &ncID})
		require.Nil(t, apiErr)
		err = storageClient.Gardens.Set(context.Background(), garden)
		require.NoError(t, err)
	})

	t.Run("WateringStarted_NotificationNotEnabled", func(t *testing.T) {
		msg := fmt.Sprintf("water,status=start,zone=0,id=eventID,zone_id=%s millis=6000", zoneID.String())
		err = handler.doWaterCompleteMessage("garden/data/water", []byte(msg))
		require.NoError(t, err)
	})

	t.Run("WateringComplete_NotificationNotEnabled", func(t *testing.T) {
		msg := fmt.Sprintf("water,status=complete,zone=0,id=eventID,zone_id=%s millis=6000", zoneID.String())
		err = handler.doWaterCompleteMessage("garden/data/water", []byte(msg))
		require.NoError(t, err)
	})

	t.Run("EnableNotifications", func(t *testing.T) {
		garden.NotificationSettings = &pkg.NotificationSettings{
			WateringStarted:  true,
			WateringComplete: true,
		}
		err = storageClient.Gardens.Set(context.Background(), garden)
		require.NoError(t, err)
	})

	t.Run("ErrorGettingZone", func(t *testing.T) {
		dneID := xid.New().String()
		msg := fmt.Appendf(nil, "water,zone=1 millis=6000 zone_id=%s id=eventID", dneID)
		err = handler.doWaterCompleteMessage("garden/data/water", msg)
		require.Error(t, err)
		require.Equal(t, fmt.Sprintf("error getting zone %s: resource not found", dneID), err.Error())
	})

	t.Run("ErrorUsingPushover", func(t *testing.T) {
		r, err := recorder.New("testdata/fixtures/pushover_fail",
			recorder.WithMatcher(urlMethodMatcher),
		)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			require.NoError(t, r.Stop())
		}()

		if r.Mode() != recorder.ModeRecordOnce {
			t.Fatal("Recorder should be in ModeRecordOnce")
		}

		http.DefaultClient = r.GetDefaultClient()

		msg := fmt.Appendf(nil, "water,zone=0 millis=6000 zone_id=%s id=eventID", zoneID.String())
		err = handler.doWaterCompleteMessage("garden/data/water", msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to send notification")
	})

	t.Run("WateringStarted_Success", func(t *testing.T) {
		numMessages := 0

		r, err := recorder.New(
			"testdata/fixtures/pushover_start_success",
			recorder.WithMatcher(urlMethodMatcher),
			recorder.WithHook(func(i *cassette.Interaction) error {
				if i.Request.URL == "https://api.pushover.net/1/messages.json" {
					numMessages++
				}
				return nil
			}, recorder.BeforeResponseReplayHook),
		)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			require.NoError(t, r.Stop())
		}()

		if r.Mode() != recorder.ModeRecordOnce {
			t.Fatal("Recorder should be in ModeRecordOnce")
		}

		http.DefaultClient = r.GetDefaultClient()

		msg := fmt.Sprintf("water,status=start,zone=0,id=eventID,zone_id=%s millis=6000", zoneID.String())
		err = handler.doWaterCompleteMessage("garden/data/water", []byte(msg))
		require.NoError(t, err)

		require.Equal(t, 1, numMessages)
	})

	t.Run("WateringComplete_Success", func(t *testing.T) {
		numMessages := 0

		r, err := recorder.New(
			"testdata/fixtures/pushover_complete_success",
			recorder.WithMatcher(urlMethodMatcher),
			recorder.WithHook(func(i *cassette.Interaction) error {
				if i.Request.URL == "https://api.pushover.net/1/messages.json" {
					numMessages++
				}
				return nil
			}, recorder.BeforeResponseReplayHook),
		)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			require.NoError(t, r.Stop())
		}()

		if r.Mode() != recorder.ModeRecordOnce {
			t.Fatal("Recorder should be in ModeRecordOnce")
		}

		http.DefaultClient = r.GetDefaultClient()

		msg := fmt.Sprintf("water,status=complete,zone=0,id=eventID,zone_id=%s millis=6000", zoneID.String())
		err = handler.doWaterCompleteMessage("garden/data/water", []byte(msg))
		require.NoError(t, err)

		require.Equal(t, 1, numMessages)
	})
}

func TestHandleMessageFake(t *testing.T) {
	storageClient, err := storage.NewClient(storage.Config{
		ConnectionString: ":memory:",
	})
	require.NoError(t, err)

	handler := NewWorker(storageClient, nil, nil, slog.Default())

	zoneID := babyapi.NewID()

	garden := &pkg.Garden{
		ID:          babyapi.NewID(),
		TopicPrefix: "garden",
		Name:        "garden",
	}
	err = storageClient.Gardens.Set(context.Background(), garden)
	require.NoError(t, err)

	zero := uint(0)
	zone := &pkg.Zone{
		ID:       zoneID,
		GardenID: garden.ID.ID,
		Position: &zero,
	}
	err = storageClient.Zones.Set(context.Background(), zone)
	require.NoError(t, err)

	nc := &notifications.Client{
		ID:  babyapi.NewID(),
		URL: "fake://",
	}
	err = storageClient.NotificationClientConfigs.Set(context.Background(), nc)
	require.NoError(t, err)

	ncID := nc.GetID()
	apiErr := garden.Patch(&pkg.Garden{NotificationClientID: &ncID})
	require.Nil(t, apiErr)

	garden.NotificationSettings = &pkg.NotificationSettings{
		WateringStarted:  true,
		WateringComplete: true,
	}
	err = storageClient.Gardens.Set(context.Background(), garden)
	require.NoError(t, err)

	t.Run("WateringStarted_FakeSuccess", func(t *testing.T) {
		defer fake_notification.Reset()

		msg := fmt.Sprintf("water,status=start,zone=0,id=eventID,zone_id=%s millis=6000", zoneID.String())
		err = handler.doWaterCompleteMessage("garden/data/water", []byte(msg))
		require.NoError(t, err)

		lastMsg := fake_notification.LastMessage()
		require.NotEmpty(t, lastMsg.Title)
	})

	t.Run("WateringComplete_FakeSuccess", func(t *testing.T) {
		defer fake_notification.Reset()

		msg := fmt.Sprintf("water,status=complete,zone=0,id=eventID,zone_id=%s millis=6000", zoneID.String())
		err = handler.doWaterCompleteMessage("garden/data/water", []byte(msg))
		require.NoError(t, err)

		lastMsg := fake_notification.LastMessage()
		require.NotEmpty(t, lastMsg.Title)
	})
}
