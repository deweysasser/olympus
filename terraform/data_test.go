package terraform

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestReadPlan(t *testing.T) {

	json, err := ReadPlan("../data/production/50-persistence/plan.json")

	assert.NoError(t, err)
	assert.NotNil(t, json)

	assert.Equal(t, 0, len(json.Variables))
	assert.False(t, json.UpToDate())
	c := json.Changes()

	assert.Equal(t, 0, c.Added)
	assert.Equal(t, 0, c.Updated)
	assert.Equal(t, 1, c.Deleted)
}
