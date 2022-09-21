package terraform

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestReadDir(t *testing.T) {
	sum, err := ReadDir("../data")

	assert.NoError(t, err)

	assert.NotNil(t, sum)

	assert.Equal(t, sum.Name(), "data")
	assert.Equal(t, 7, len(sum.Children()))
}
