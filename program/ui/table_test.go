package ui

import (
	"github.com/deweysasser/olympus/terraform"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCreateTable(t *testing.T) {
	data, err := terraform.ReadDir("../../data")
	if err != nil {
		t.Fatal("Failed to read data")
	}

	tab := CreateTable(data.Children())

	assert.Equal(t, []string{"billing", "devlab", "production", "production-reporting", "security", "staging", "staging-topaz-a"}, tab.Columns)
	var names []string
	for _, row := range tab.Rows {
		names = append(names, string(row.Name))
	}
	assert.Equal(t, []string{"10-bootstrap", "20-domains", "20-user", "30-network", "35-interconnect", "40-infrastructure", "50-persistence", "60-service", "70-application", "80-control", "90-publish", "91-tracking", "93-monitoring", "95-legacy", "account"},
		names)
}
