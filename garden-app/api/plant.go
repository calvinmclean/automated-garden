package api

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"gopkg.in/yaml.v2"
)

// Plant is the representation of the most important resource for this application, a Plant.
// This includes some general information like name and unique ID, a start and end date to show when
// the Plant was in the system, plus some information for watering like the duration to water for, how
// often to water, and the PlantPosition field will tell the microcontroller which plant to water
type Plant struct {
	Name           string `json:"name" yaml:"name"`
	ID             string `json:"id" yaml:"id"`
	WateringAmount int    `json:"watering_amount" yaml:"watering_amount"`
	PlantPosition  int    `json:"plant_position" yaml:"plant_position"`
	Interval       string `json:"interval" yaml:"interval"`
	StartDate      string `json:"start_date" yaml:"start_date"`
	EndDate        string `json:"end_date" yaml:"end_date"`
}

// Render is used to make this struct compatible with the go-chi webserver for writing
// the JSON response
func (p *Plant) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// Bind is used to make this struct compatible with the go-chi webserver for reading incoming
// JSON requests
func (p *Plant) Bind(r *http.Request) error {
	if p == nil {
		return errors.New("missing required action fields")
	}

	return nil
}

// ReadPlants will read a map of Plants from a YAML file
func ReadPlants(filename string) map[string]*Plant {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Print(err)
	}
	var plants map[string]*Plant
	err = yaml.Unmarshal(data, &plants)
	if err != nil {
		fmt.Println("error:", err)
	}
	return plants
}