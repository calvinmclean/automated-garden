package hashmap

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// ByteSlice is a struct that is used to implement custom YAML Marshal/Unmarshal because
// the default marshal for a []byte will write an array of integers
type ByteSlice []byte

// MarshalYAML simply casts the ByteSlice to a string
func (bs ByteSlice) MarshalYAML() (interface{}, error) {
	return string(bs), nil
}

// UnmarshalYAML converts the YAML string back into ByteSlice
func (bs *ByteSlice) UnmarshalYAML(value *yaml.Node) error {
	switch value.Tag {
	case "!!str":
		*bs = []byte(value.Value)
	default:
		return fmt.Errorf("expected string, but got %s", value.Tag)
	}
	return nil
}
