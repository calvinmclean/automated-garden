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
	data          clientData
	k8sClient     v1.ConfigMapInterface
	Options       map[string]string

	m *sync.Mutex
}

type clientData struct {
	Gardens              map[xid.ID]*pkg.Garden     `yaml:"gardens"`
	WeatherClientConfigs map[xid.ID]*weather.Config `yaml:"weather_clients"`
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
		data: clientData{
			Gardens:              map[xid.ID]*pkg.Garden{},
			WeatherClientConfigs: map[xid.ID]*weather.Config{},
		},
		Options: options,
		m:       &sync.Mutex{},
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

// save saves the client's data back to a persistent source. This is unexported and should only be used when a RWLock is already acquired
func (c *Client) save() error {
	// Marshal map to YAML bytes
	content, err := yaml.Marshal(c.data)
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
	err = yaml.Unmarshal([]byte(configMap.Data[c.keyName]), &c.data)
	if err != nil {
		return fmt.Errorf("unable to unmarshal YAML: %v", err)
	}
	return nil
}
