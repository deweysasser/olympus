package terraform

import (
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestReadPlan(t *testing.T) {

	var json PlanSummary

	json = &JSonPlanSummary{
		Plan: &tfjson.Plan{
			ResourceChanges: []*tfjson.ResourceChange{
				&tfjson.ResourceChange{Change: &tfjson.Change{
					Actions: []tfjson.Action{tfjson.ActionDelete},
				}},
			},
		},
		name: "some/path",
	}

	assert.False(t, json.UpToDate())
	c := json.Changes()

	assert.Equal(t, 0, c.ResourcesAdded)
	assert.Equal(t, 0, c.ResourcesUpdated)
	assert.Equal(t, 1, c.ResourcesDeleted)

	assert.True(t, c.HasAny())
	assert.Equal(t, "deleted", c.Highest())
}
