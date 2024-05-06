package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/notifications"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/babyapi"
	babytest "github.com/calvinmclean/babyapi/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationClientAPI(t *testing.T) {
	storageClient, err := storage.NewClient(storage.Config{
		Driver: "hashmap",
	})
	require.NoError(t, err)

	api := NewNotificationClientsAPI()
	api.setup(storageClient)

	babytest.RunTableTest(t, api.API, []babytest.TestCase[*babyapi.AnyResource]{
		{
			Name: "CreateFakeClient",
			Test: babytest.RequestTest[*babyapi.AnyResource]{
				Method: http.MethodPost,
				Body:   `{"name": "fake client", "type": "fake", "options": {}}`,
			},
			ExpectedResponse: babytest.ExpectedResponse{
				Status:     http.StatusCreated,
				BodyRegexp: `{"id":"[0-9a-v]{20}","name":"fake client","type":"fake","options":{},"links":\[{"rel":"self","href":"/notification_clients/[0-9a-v]{20}"}\]}`,
			},
		},
		{
			Name: "GetFakeClient",
			Test: babytest.RequestTest[*babyapi.AnyResource]{
				Method: http.MethodGet,
				IDFunc: func(getResponse babytest.PreviousResponseGetter) string {
					return getResponse("CreateFakeClient").Data.GetID()
				},
			},
			ExpectedResponse: babytest.ExpectedResponse{
				Status:     http.StatusOK,
				BodyRegexp: `{"id":"[0-9a-v]{20}","name":"fake client","type":"fake","options":{},"links":\[{"rel":"self","href":"/notification_clients/[0-9a-v]{20}"}\]}`,
			},
		},
		{
			Name: "ErrorCreateFakeClient",
			Test: babytest.RequestTest[*babyapi.AnyResource]{
				Method: http.MethodPost,
				Body:   `{"name": "fake client", "type": "fake", "options": {"create_error": "fail!"}}`,
			},
			ExpectedResponse: babytest.ExpectedResponse{
				Status: http.StatusBadRequest,
				Error:  `error posting resource: unexpected response with text: Invalid request.`,
				Body:   `{"status":"Invalid request.","error":"error initializing client: fail!"}`,
			},
		},
		{
			Name: "CreateClientErrorNoName",
			Test: babytest.RequestTest[*babyapi.AnyResource]{
				Method: http.MethodPost,
				Body:   `{"type": "fake", "options": {}}`,
			},
			ExpectedResponse: babytest.ExpectedResponse{
				Status: http.StatusBadRequest,
				Error:  `error posting resource: unexpected response with text: Invalid request.`,
				Body:   `{"status":"Invalid request.","error":"missing required name field"}`,
			},
		},
		{
			Name: "CreateClientErrorNoType",
			Test: babytest.RequestTest[*babyapi.AnyResource]{
				Method: http.MethodPost,
				Body:   `{"name": "fake client", "options": {}}`,
			},
			ExpectedResponse: babytest.ExpectedResponse{
				Status: http.StatusBadRequest,
				Error:  `error posting resource: unexpected response with text: Invalid request.`,
				Body:   `{"status":"Invalid request.","error":"missing required type field"}`,
			},
		},
		{
			Name: "CreateClientErrorNoOptions",
			Test: babytest.RequestTest[*babyapi.AnyResource]{
				Method: http.MethodPost,
				Body:   `{"name": "fake client", "type": "fake"}`,
			},
			ExpectedResponse: babytest.ExpectedResponse{
				Status: http.StatusBadRequest,
				Error:  `error posting resource: unexpected response with text: Invalid request.`,
				Body:   `{"status":"Invalid request.","error":"missing required options field"}`,
			},
		},
	})

	t.Run("SendTestMessage", func(t *testing.T) {
		var client notifications.Client
		t.Run("CreateClient", func(t *testing.T) {
			r := httptest.NewRequest(
				http.MethodPost,
				"/notification_clients",
				strings.NewReader(`{"name": "fake client", "type": "fake", "options": {}}`),
			)
			r.Header.Add("Content-Type", "application/json")
			w := babytest.TestRequest(t, api.API, r)
			assert.Equal(t, http.StatusCreated, w.Code)

			err = json.NewDecoder(w.Body).Decode(&client)
			require.NoError(t, err)
		})

		t.Run("SendTestMessage", func(t *testing.T) {
			r := httptest.NewRequest(
				http.MethodPost,
				fmt.Sprintf("/notification_clients/%s/test", client.GetID()),
				strings.NewReader(`{"title":"test title","message":"test message"}`),
			)
			r.Header.Add("Content-Type", "application/json")
			w := babytest.TestRequest(t, api.API, r)
			assert.Equal(t, http.StatusOK, w.Code)
		})

		var errorClient notifications.Client
		t.Run("CreateErrorClient", func(t *testing.T) {
			r := httptest.NewRequest(
				http.MethodPost,
				"/notification_clients",
				strings.NewReader(`{"name": "fake client", "type": "fake", "options": {"send_message_error": "fail!"}}`),
			)
			r.Header.Add("Content-Type", "application/json")
			w := babytest.TestRequest(t, api.API, r)
			assert.Equal(t, http.StatusCreated, w.Code)

			err = json.NewDecoder(w.Body).Decode(&errorClient)
			require.NoError(t, err)
		})

		t.Run("SendTestMessageError", func(t *testing.T) {
			r := httptest.NewRequest(
				http.MethodPost,
				fmt.Sprintf("/notification_clients/%s/test", errorClient.GetID()),
				strings.NewReader(`{"title":"test title","message":"test message"}`),
			)
			r.Header.Add("Content-Type", "application/json")
			w := babytest.TestRequest(t, api.API, r)
			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	})
}
