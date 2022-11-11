package netatmo

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWeatherDataUnmarshalJSON(t *testing.T) {
	input := `{
	"body": {
		"1662058800":[
			7.9
		],
		"1662145200":[
			2.5
		],
		"1662231600":[
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
