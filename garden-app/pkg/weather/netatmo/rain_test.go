package netatmo

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshalJSON(t *testing.T) {
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
	var rainDataResp rainDataResponse
	err := json.Unmarshal([]byte(input), &rainDataResp)
	assert.NoError(t, err)
	assert.Equal(t, rainData{
		time.Unix(1662058800, 0): 7.9,
		time.Unix(1662145200, 0): 2.5,
		time.Unix(1662231600, 0): 0,
	}, rainDataResp.Body)
}

func TestTotal(t *testing.T) {
	data := rainData{
		time.Unix(1662058800, 0): 7.9,
		time.Unix(1662145200, 0): 2.5,
		time.Unix(1662231600, 0): 0,
	}
	assert.Equal(t, float32(10.4), data.Total())
}
