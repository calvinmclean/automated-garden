package pkg

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Duration is a wrapper around the time.Duration that allows it to be JSON Unmarshalled
// as a string or int64 and then Marshaled as an int64
type Duration struct {
	time.Duration
}

// MarshalJSON will convert Duration into the string representation
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// UnmarshalJSON with allow reading a Duration as a string or integer into time.Duration
func (d *Duration) UnmarshalJSON(data []byte) error {
	var value interface{}
	err := json.Unmarshal(data, &value)
	if err != nil {
		return err
	}

	switch v := value.(type) {
	case string:
		d.Duration, err = time.ParseDuration(strings.Trim(v, `"`))
		if err != nil {
			return fmt.Errorf("invalid format for Duration: %w", err)
		}
	case float64:
		d.Duration = time.Duration(v)
	default:
		return fmt.Errorf("unexpected type %T, must be string or number", v)
	}

	return nil
}

// UnmarshalYAML with allow reading a Duration as a string or integer into time.Duration
func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	switch value.Tag {
	case "!!str":
		var err error
		d.Duration, err = time.ParseDuration(strings.Trim(value.Value, `"`))
		if err != nil {
			return fmt.Errorf("invalid format for Duration: %w", err)
		}
	case "!!int":
		v, err := strconv.Atoi(value.Value)
		if err != nil {
			return err
		}
		d.Duration = time.Duration(v)
	default:
		return fmt.Errorf("unexpected type %s, must be string or number", value.Tag)
	}

	return nil
}

// MarshalYAML will convert Duration into the string representation
func (d *Duration) MarshalYAML() (interface{}, error) {
	return d.String(), nil
}
