package storage

import (
	"context"
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/rs/xid"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

// ConfigMapClient implements the Client interface to use a Kubernetes ConfigMap to
// store a YAML file containing Plant information
type ConfigMapClient struct {
	configMapName string
	keyName       string
	gardens       map[xid.ID]*pkg.Garden
	k8sClient     v1.ConfigMapInterface
	Config        Config
}

// NewConfigMapClient initializes a K8s clientset and reads the ConfigMap into a map
func NewConfigMapClient(config Config) (*ConfigMapClient, error) {
	if _, ok := config.Options["name"]; !ok {
		return nil, fmt.Errorf("missing config key 'name'")
	}
	if _, ok := config.Options["key"]; !ok {
		return nil, fmt.Errorf("missing config key 'key'")
	}
	client := &ConfigMapClient{
		configMapName: config.Options["name"],
		keyName:       config.Options["key"],
		gardens:       map[xid.ID]*pkg.Garden{},
		Config:        config,
	}

	// Ccreate the in-cluster config
	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("unable to create InClusterConfig: %v", err)
	}
	// Create the clientset
	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create Clientset: %v", err)
	}

	client.k8sClient = clientset.CoreV1().ConfigMaps("default")

	// Get the ConfigMap and read into map
	err = client.update()
	if err != nil {
		return client, err
	}

	return client, nil
}

func (c *ConfigMapClient) update() error {
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

// GetGarden returns the garden
func (c *ConfigMapClient) GetGarden(id xid.ID) (*pkg.Garden, error) {
	err := c.update()
	if err != nil {
		return nil, err
	}
	return c.gardens[id], nil
}

// GetGardens returns all gardens
func (c *ConfigMapClient) GetGardens(getEndDated bool) ([]*pkg.Garden, error) {
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
func (c *ConfigMapClient) SaveGarden(garden *pkg.Garden) error {
	c.gardens[garden.ID] = garden
	return c.Save()
}

// DeleteGarden permanently deletes a garden and removes it from the YAML file
func (c *ConfigMapClient) DeleteGarden(garden xid.ID) error {
	delete(c.gardens, garden)
	return c.Save()
}

// GetZone just returns the request Zone from the map
func (c *ConfigMapClient) GetZone(garden xid.ID, id xid.ID) (*pkg.Zone, error) {
	err := c.update()
	if err != nil {
		return nil, err
	}
	return c.gardens[garden].Zones[id], nil
}

// GetZones returns all zones from the map as a slice
func (c *ConfigMapClient) GetZones(garden xid.ID, getEndDated bool) ([]*pkg.Zone, error) {
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
func (c *ConfigMapClient) SaveZone(gardenID xid.ID, zone *pkg.Zone) error {
	if c.gardens[gardenID].Zones == nil {
		c.gardens[gardenID].Zones = map[xid.ID]*pkg.Zone{}
	}
	c.gardens[gardenID].Zones[zone.ID] = zone
	return c.Save()
}

// DeleteZone permanently deletes a zone and removes it from the YAML file
func (c *ConfigMapClient) DeleteZone(garden xid.ID, zone xid.ID) error {
	delete(c.gardens[garden].Zones, zone)
	return c.Save()
}

// GetPlant just returns the request Plant from the map
func (c *ConfigMapClient) GetPlant(garden xid.ID, id xid.ID) (*pkg.Plant, error) {
	err := c.update()
	if err != nil {
		return nil, err
	}
	return c.gardens[garden].Plants[id], nil
}

// GetPlants returns all plants from the map as a slice
func (c *ConfigMapClient) GetPlants(garden xid.ID, getEndDated bool) ([]*pkg.Plant, error) {
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
func (c *ConfigMapClient) SavePlant(gardenID xid.ID, plant *pkg.Plant) error {
	if c.gardens[gardenID].Plants == nil {
		c.gardens[gardenID].Plants = map[xid.ID]*pkg.Plant{}
	}
	c.gardens[gardenID].Plants[plant.ID] = plant
	return c.Save()
}

// DeletePlant permanently deletes a plant and removes it from the YAML file
func (c *ConfigMapClient) DeletePlant(garden xid.ID, plant xid.ID) error {
	delete(c.gardens[garden].Plants, plant)
	return c.Save()
}

// Save saves the client's data back to a persistent source
func (c *ConfigMapClient) Save() error {
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
