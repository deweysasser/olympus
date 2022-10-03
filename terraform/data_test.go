package terraform

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDataMarshal(t *testing.T) {
	sum := &SummaryData{Name: "testing"}

	b, e := json.Marshal(&sum)
	assert.NoError(t, e)
	assert.Equal(t, "{\"name\":\"testing\",\"changes\":{\"added\":0,\"updated\":0,\"deleted\":0}}", string(b))
}
