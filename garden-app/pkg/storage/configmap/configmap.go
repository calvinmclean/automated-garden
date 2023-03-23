package configmap

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/rs/xid"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

// Client implements the Client interface to use a Kubernetes ConfigMap to
// store a YAML file containing Plant information
type Client struct {
	configMapName string
	keyName       string
	gardens       map[xid.ID]*pkg.Garden
	k8sClient     v1.ConfigMapInterface
	Options       map[string]string

	m *sync.Mutex
}

// NewClient initializes a K8s clientset and reads the ConfigMap into a map
func NewClient(options map[string]string) (*Client, error) {
	if _, ok := options["name"]; !ok {
		return nil, fmt.Errorf("missing config key 'name'")
	}
	if _, ok := options["key"]; !ok {
		return nil, fmt.Errorf("missing config key 'key'")
	}
	client := &Client{
		configMapName: options["name"],
		keyName:       options["key"],
		gardens:       map[xid.ID]*pkg.Garden{},
		Options:       options,
		m:             &sync.Mutex{},
	}

	// Create the in-cluster config
	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("unable to create InClusterConfig: %v", err)
	}
	// Create the clientset
	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create Clientset: %v", err)
	}

	namespace, err := os.ReadFile("/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return nil, fmt.Errorf("unable to read namespace from file: %v", err)
	}

	client.k8sClient = clientset.CoreV1().ConfigMaps(string(namespace))

	// Get the ConfigMap and read into map
	err = client.update()
	if err != nil {
		return client, err
	}

	return client, nil
}

// GetGarden returns the garden
func (c *Client) GetGarden(id xid.ID) (*pkg.Garden, error) {
	c.m.Lock()
	defer c.m.Unlock()

	err := c.update()
	if err != nil {
		return nil, err
	}
	return c.gardens[id], nil
}

// GetGardens returns all gardens
func (c *Client) GetGardens(getEndDated bool) ([]*pkg.Garden, error) {
	c.m.Lock()
	defer c.m.Unlock()

	err := c.update()
	if err != nil {
		return nil, err
	}

	result := []*pkg.Garden{}
	for _, g := range c.gardens {
		if getEndDated || !g.EndDated() {
			result = append(result, g)
		}
	}
	return result, nil
}

// SaveGarden saves a garden and writes it back to the ConfigMap
func (c *Client) SaveGarden(garden *pkg.Garden) error {
	c.m.Lock()
	defer c.m.Unlock()

	c.gardens[garden.ID] = garden
	return c.save()
}

// DeleteGarden permanently deletes a garden and removes it from the YAML file
func (c *Client) DeleteGarden(garden xid.ID) error {
	c.m.Lock()
	defer c.m.Unlock()

	delete(c.gardens, garden)
	return c.save()
}

// GetZone just returns the request Zone from the map
func (c *Client) GetZone(garden xid.ID, id xid.ID) (*pkg.Zone, error) {
	c.m.Lock()
	defer c.m.Unlock()

	err := c.update()
	if err != nil {
		return nil, err
	}
	return c.gardens[garden].Zones[id], nil
}

// GetZones returns all zones from the map as a slice
func (c *Client) GetZones(garden xid.ID, getEndDated bool) ([]*pkg.Zone, error) {
	c.m.Lock()
	defer c.m.Unlock()

	err := c.update()
	if err != nil {
		return nil, err
	}
	result := []*pkg.Zone{}
	for _, p := range c.gardens[garden].Zones {
		if getEndDated || !p.EndDated() {
			result = append(result, p)
		}
	}
	return result, nil
}

// SaveZone saves a zone in the map and will write it back to the YAML file
func (c *Client) SaveZone(gardenID xid.ID, zone *pkg.Zone) error {
	c.m.Lock()
	defer c.m.Unlock()

	if c.gardens[gardenID].Zones == nil {
		c.gardens[gardenID].Zones = map[xid.ID]*pkg.Zone{}
	}
	c.gardens[gardenID].Zones[zone.ID] = zone
	return c.save()
}

// DeleteZone permanently deletes a zone and removes it from the YAML file
func (c *Client) DeleteZone(garden xid.ID, zone xid.ID) error {
	c.m.Lock()
	defer c.m.Unlock()

	delete(c.gardens[garden].Zones, zone)
	return c.save()
}

// GetPlant just returns the request Plant from the map
func (c *Client) GetPlant(garden xid.ID, id xid.ID) (*pkg.Plant, error) {
	c.m.Lock()
	defer c.m.Unlock()

	err := c.update()
	if err != nil {
		return nil, err
	}
	return c.gardens[garden].Plants[id], nil
}

// GetPlants returns all plants from the map as a slice
func (c *Client) GetPlants(garden xid.ID, getEndDated bool) ([]*pkg.Plant, error) {
	c.m.Lock()
	defer c.m.Unlock()

	err := c.update()
	if err != nil {
		return nil, err
	}
	result := []*pkg.Plant{}
	for _, p := range c.gardens[garden].Plants {
		if getEndDated || !p.EndDated() {
			result = append(result, p)
		}
	}
	return result, nil
}

// SavePlant saves a plant in the map and will write it back to the YAML file
func (c *Client) SavePlant(gardenID xid.ID, plant *pkg.Plant) error {
	c.m.Lock()
	defer c.m.Unlock()

	if c.gardens[gardenID].Plants == nil {
		c.gardens[gardenID].Plants = map[xid.ID]*pkg.Plant{}
	}
	c.gardens[gardenID].Plants[plant.ID] = plant
	return c.save()
}

// DeletePlant permanently deletes a plant and removes it from the YAML file
func (c *Client) DeletePlant(garden xid.ID, plant xid.ID) error {
	c.m.Lock()
	defer c.m.Unlock()

	delete(c.gardens[garden].Plants, plant)
	return c.save()
}

// save saves the client's data back to a persistent source. This is unexported and should only be used when a RWLock is already acquired
func (c *Client) save() error {
	// Marshal map to YAML bytes
	content, err := yaml.Marshal(c.gardens)
	if err != nil {
		return fmt.Errorf("unable to marshal YAML string from Plants map: %v", err)
	}

	// Read the current ConfigMap, overwrite the Plants data, and update it
	configMap, err := c.k8sClient.Get(context.TODO(), c.configMapName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("unable to get ConfigMap '%s' from K8s: %v", c.configMapName, err)
	}
	configMap.Data[c.keyName] = string(content)
	_, err = c.k8sClient.Update(context.TODO(), configMap, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("unable to update ConfigMap '%s' in K8s cluster: %v", c.configMapName, err)
	}
	return nil
}

// update will refresh from the configmap in case something was changed externally. Although it is mostly used prior to reads, it
// still modifies the map and should only be used while an RWLock is acquired
func (c *Client) update() error {
	configMap, err := c.k8sClient.Get(context.TODO(), c.configMapName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("unable to get ConfigMap '%s': %v", c.configMapName, err)
	}
	err = yaml.Unmarshal([]byte(configMap.Data[c.keyName]), &c.gardens)
	if err != nil {
		return fmt.Errorf("unable to unmarshal YAML map of Plants: %v", err)
	}
	return nil
}

func (c *Client) GetWeatherClient(xid.ID) (weather.Client, error)        { return nil, nil }
func (c *Client) GetWeatherClientConfig(xid.ID) (*weather.Config, error) { return nil, nil }
func (c *Client) GetWeatherClientConfigs() ([]*weather.Config, error)    { return nil, nil }
func (c *Client) SaveWeatherClientConfig(*weather.Config) error          { return nil }
func (c *Client) DeleteWeatherClientConfig(xid.ID) error                 { return nil }
