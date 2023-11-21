package storage

import (
	"errors"
	"testing"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
)

var id, _ = xid.FromString("c5cvhpcbcv45e8bp16dg")

func unmarshalError(_ []byte, _ interface{}) error {
	return errors.New("unmarshal error")
}

func marshalError(_ interface{}) ([]byte, error) {
	return nil, errors.New("marshal error")
}

func TestGetOneErrors(t *testing.T) {
	c, err := NewClient(Config{Driver: "hashmap"})
	assert.NoError(t, err)

	t.Run("ErrorNilKey", func(t *testing.T) {
		_, err := GetOne[*pkg.Garden](c, "")
		assert.Error(t, err)
		assert.Equal(t, "error getting data: Key cannot be nil", err.Error())
	})

	t.Run("ErrorUnmarshal", func(t *testing.T) {
		c.unmarshal = unmarshalError

		err := Save[*pkg.Garden](c, &pkg.Garden{ID: id}, gardenKey(id))
		assert.NoError(t, err)

		_, err = GetOne[*pkg.Garden](c, gardenKey(id))
		assert.Error(t, err)
		assert.Equal(t, "error parsing data: unmarshal error", err.Error())
	})
}

func TestGetMultipleErrors(t *testing.T) {
	c, err := NewClient(Config{Driver: "hashmap"})
	assert.NoError(t, err)

	t.Run("ErrorUnmarshal", func(t *testing.T) {
		c.unmarshal = unmarshalError

		err := Save[*pkg.Garden](c, &pkg.Garden{ID: id}, gardenKey(id))
		assert.NoError(t, err)

		_, err = GetMultiple[*pkg.Garden](c, true, gardenPrefix)
		assert.Error(t, err)
		assert.Equal(t, "error getting data: error parsing data: unmarshal error", err.Error())
	})
}

func TestSaveErrors(t *testing.T) {
	c, err := NewClient(Config{Driver: "hashmap"})
	assert.NoError(t, err)

	t.Run("ErrorNilKey", func(t *testing.T) {
		err := Save[*pkg.Garden](c, &pkg.Garden{}, "")
		assert.Error(t, err)
		assert.Equal(t, "error writing data to database: Key cannot be nil", err.Error())
	})

	t.Run("ErrorMarshal", func(t *testing.T) {
		c.marshal = marshalError

		err := Save[*pkg.Garden](c, &pkg.Garden{ID: id}, gardenKey(id))
		assert.Error(t, err)
		assert.Equal(t, "error marshalling data: marshal error", err.Error())
	})
}
