package storage

import (
	"errors"
	"testing"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/babyapi"
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
	bc, err := newBaseClient(Config{Driver: "hashmap"})
	assert.NoError(t, err)
	c := newGenericClient[*pkg.Garden](bc, "Garden")

	t.Run("ErrorUnmarshal", func(t *testing.T) {
		c.unmarshal = unmarshalError

		err := c.Set(&pkg.Garden{ID: babyapi.ID{ID: id}})
		assert.NoError(t, err)

		_, err = c.Get(id.String())
		assert.Error(t, err)
		assert.Equal(t, "error parsing data: unmarshal error", err.Error())
	})
}

func TestGetMultipleErrors(t *testing.T) {
	bc, err := newBaseClient(Config{Driver: "hashmap"})
	assert.NoError(t, err)
	c := newGenericClient[*pkg.Garden](bc, "Garden")

	t.Run("ErrorUnmarshal", func(t *testing.T) {
		c.unmarshal = unmarshalError

		err := c.Set(&pkg.Garden{ID: babyapi.ID{ID: id}})
		assert.NoError(t, err)

		_, err = c.GetAll(func(g *pkg.Garden) bool {
			return true
		})
		assert.Error(t, err)
		assert.Equal(t, "error getting data: error parsing data: unmarshal error", err.Error())
	})
}

func TestSaveErrors(t *testing.T) {
	bc, err := newBaseClient(Config{Driver: "hashmap"})
	assert.NoError(t, err)
	c := newGenericClient[*pkg.Garden](bc, "Garden")

	t.Run("ErrorMarshal", func(t *testing.T) {
		c.marshal = marshalError

		err := c.Set(&pkg.Garden{ID: babyapi.ID{ID: id}})
		assert.Error(t, err)
		assert.Equal(t, "error marshalling data: marshal error", err.Error())
	})
}
