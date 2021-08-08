package storage

import (
	"context"
	"fmt"
	"time"

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
	gardens       map[string]*pkg.Garden
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
		gardens:       map[string]*pkg.Garden{},
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
	configMap, err := client.k8sClient.Get(context.TODO(), client.configMapName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to get ConfigMap '%s': %v", client.configMapName, err)
	}
	err = yaml.Unmarshal([]byte(configMap.Data[client.keyName]), &client.gardens)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal YAML map of Plants: %v", err)
	}

	// Create start dates for Gardens and Plants if it is empty
	for _, garden := range client.gardens {
		now := time.Now().Add(1 * time.Minute)
		if garden.CreatedAt == nil {
			garden.CreatedAt = &now
			client.Save()
		}
		for _, plant := range garden.Plants {
			if plant.CreatedAt == nil {
				now := time.Now().Add(1 * time.Minute)
				plant.CreatedAt = &now
				client.SavePlant(garden.Name, plant)
			}
		}
	}

	return client, nil
}

// GetGarden returns the garden
func (c *ConfigMapClient) GetGarden(name string) (*pkg.Garden, error) {
	return c.gardens[name], nil
}

// GetGardens returns all gardens
func (c *ConfigMapClient) GetGardens(getEndDated bool) ([]*pkg.Garden, error) {
	result := []*pkg.Garden{}
	for _, g := range c.gardens {
		if getEndDated || (!getEndDated && g.EndDate == nil) {
			result = append(result, g)
		}
	}
	return result, nil
}

// GetPlant just returns the request Plant from the map
func (c *ConfigMapClient) GetPlant(garden string, id xid.ID) (*pkg.Plant, error) {
	return c.gardens[garden].Plants[id], nil
}

// GetPlants returns all plants from the map as a slice
func (c *ConfigMapClient) GetPlants(garden string, getEndDated bool) ([]*pkg.Plant, error) {
	result := []*pkg.Plant{}
	for _, p := range c.gardens[garden].Plants {
		// Only return end-dated plants if specifically asked for
		if getEndDated || (!getEndDated && p.EndDate == nil) {
			result = append(result, p)
		}
	}
	return result, nil
}

// SavePlant saves a plant in the map and will write it back to the YAML file
func (c *ConfigMapClient) SavePlant(garden string, plant *pkg.Plant) error {
	c.gardens[garden].Plants[plant.ID] = plant
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
