package netatmo

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWeatherDataUnmarshalJSON_NewFormat(t *testing.T) {
	input := `{
	  "body": [
	    {
	      "beg_time": 1786302000,
	      "value": [
	        [
	          42.7
	        ]
	      ]
	    }
	  ],
	  "status": "ok",
	  "time_exec": 0.025828838348388672,
	  "time_server": 1786301575
	}`
	var weatherDataResp weatherDataResponse
	err := json.Unmarshal([]byte(input), &weatherDataResp)
	assert.NoError(t, err)
	assert.Equal(t, weatherData{
		time.Unix(1786302000, 0): 42.7,
	}, weatherDataResp.Body)
}

func TestWeatherDataUnmarshalJSON_OldFormat(t *testing.T) {
	input := `{
	  "body": {
	    "1662058800": [
	      7.9
	    ],
	    "1662145200": [
	      2.5
	    ],
	    "1662231600": [
	      0
	    ]
	  },
	  "status": "ok",
	  "time_exec": 0.08324098587036133,
	  "time_server": 1662249790
	}`
	var weatherDataResp weatherDataResponse
	err := json.Unmarshal([]byte(input), &weatherDataResp)
	assert.NoError(t, err)
	assert.Equal(t, weatherData{
		time.Unix(1662058800, 0): 7.9,
		time.Unix(1662145200, 0): 2.5,
		time.Unix(1662231600, 0): 0,
	}, weatherDataResp.Body)
}

func TestWeatherDataTotal(t *testing.T) {
	data := weatherData{
		time.Unix(1662058800, 0): 7.9,
		time.Unix(1662145200, 0): 2.5,
		time.Unix(1662231600, 0): 0,
	}
	assert.Equal(t, float32(10.4), data.Total())
}

func TestWeatherDataAverage(t *testing.T) {
	t.Run("WithData", func(t *testing.T) {
		data := weatherData{
			time.Unix(1662058800, 0): 7.9,
			time.Unix(1662145200, 0): 2.5,
			time.Unix(1662231600, 0): 0,
		}
		assert.InDelta(t, float32(3.4666), data.Average(), 0.001)
	})

	t.Run("Empty", func(t *testing.T) {
		data := weatherData{}
		assert.Equal(t, float32(0), data.Average())
	})
}

func TestWeatherDataIsEmpty(t *testing.T) {
	empty := weatherData{}
	assert.True(t, empty.IsEmpty())

	data := weatherData{time.Unix(1662058800, 0): 7.9}
	assert.False(t, data.IsEmpty())
}
