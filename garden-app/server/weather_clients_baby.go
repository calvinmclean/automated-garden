package server

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/calvinmclean/babyapi"
	"github.com/rs/xid"
)

type WeatherConfig struct {
	*weather.Config

	Links []Link `json:"links,omitempty"`
}

func (wc *WeatherConfig) GetID() string {
	return wc.ID.String()
}

func (wc *WeatherConfig) Render(_ http.ResponseWriter, _ *http.Request) error {
	if wc != nil {
		wc.Links = append(wc.Links,
			Link{
				"self",
				fmt.Sprintf("%s/%s", weatherClientsBasePath, wc.ID),
			},
		)
	}

	return nil
}

func (wc *WeatherConfig) Bind(r *http.Request) error {
	if wc == nil || wc.Config == nil {
		return errors.New("missing required WeatherClient fields")
	}

	switch r.Method {
	case http.MethodPost:
		if wc.Config.Type == "" {
			return errors.New("missing required type field")
		}
		if wc.Config.Options == nil {
			return errors.New("missing required options field")
		}
		_, err := weather.NewClient(wc.Config, func(map[string]interface{}) error { return nil })
		if err != nil {
			return fmt.Errorf("failed to create valid client using config: %w", err)
		}

	case http.MethodPatch:
		if wc.ID != xid.NilID() {
			return errors.New("updating ID is not allowed")
		}
	}

	return nil
}

type WeatherStorageClient struct {
	sc *storage.Client
}

var _ babyapi.Storage[*WeatherConfig] = &WeatherStorageClient{}

func (wsc *WeatherStorageClient) Get(id string) (*WeatherConfig, error) {
	result, err := storage.GetOne[WeatherConfig](wsc.sc, storage.WeatherClientKey(id))
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, babyapi.ErrNotFound
	}

	return result, nil
}

func (wsc *WeatherStorageClient) GetAll() ([]*WeatherConfig, error) {
	return storage.GetMultiple[*WeatherConfig](wsc.sc, true, "WeatherClient_")
}

func (wsc *WeatherStorageClient) Set(wc *WeatherConfig) error {
	return storage.Save[*WeatherConfig](wsc.sc, wc, storage.WeatherClientKey(wc.ID.String()))
}

func (wsc *WeatherStorageClient) Delete(id string) error {
	return storage.Delete(wsc.sc, storage.WeatherClientKey(id))
}
