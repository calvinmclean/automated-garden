package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/api"
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
	plants        map[xid.ID]*api.Plant
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
		Config:        config,
	}

	// Ccreate the in-cluster config
	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	// Create the clientset
	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, err
	}

	client.k8sClient = clientset.CoreV1().ConfigMaps("default")

	// Get the ConfigMap and read into map
	configMap, err := client.k8sClient.Get(context.TODO(), client.configMapName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal([]byte(configMap.Data[client.keyName]), &client.plants)
	if err != nil {
		return nil, err
	}

	// Create start dates for Plants if it is empty
	for _, plant := range client.plants {
		if plant.StartDate == nil {
			now := time.Now().Add(1 * time.Minute)
			plant.StartDate = &now
			client.SavePlant(plant)
		}
	}

	return client, nil
}

// GetPlant just returns the request Plant from the map
func (c *ConfigMapClient) GetPlant(id xid.ID) (*api.Plant, error) {
	return c.plants[id], nil
}

// GetPlants returns all plants from the map as a slice
func (c *ConfigMapClient) GetPlants(getEndDated bool) []*api.Plant {
	result := []*api.Plant{}
	for _, p := range c.plants {
		// Only return end-dated plants if specifically asked for
		if getEndDated || (!getEndDated && p.EndDate == nil) {
			result = append(result, p)
		}
	}
	return result
}

// SavePlant saves a plant in the map and will write it back to the YAML file
func (c *ConfigMapClient) SavePlant(plant *api.Plant) error {
	c.plants[plant.ID] = plant

	// Marshal map to YAML bytes
	content, err := yaml.Marshal(c.plants)
	if err != nil {
		return err
	}

	// Read the current ConfigMap, overwrite the Plants data, and update it
	configMap, err := c.k8sClient.Get(context.TODO(), c.configMapName, metav1.GetOptions{})
	configMap.Data[c.keyName] = string(content)
	_, err = c.k8sClient.Update(context.TODO(), configMap, metav1.UpdateOptions{})
	return err
}
