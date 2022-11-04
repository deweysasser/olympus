package terraform

import (
	"encoding/json"
	"fmt"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/pkg/errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type PlanSummary interface {
	// Name is the name of the grouping (environment, plane, whatever)
	Name() string
	// Changes calculates the number of additions, changes, and deletions
	Changes() Changes
	UpToDate() bool
	Children() []PlanSummary
	ChangedResources() string
}

func (j *JSonPlanSummary) Children() []PlanSummary {
	return []PlanSummary{}
}

// JSonPlanSummary is a summary based on the actual terraform plan
type JSonPlanSummary struct {
	*tfjson.Plan
	name string
}

func (j *JSonPlanSummary) ChangedResources() string {
	var resources []string

	for _, rc := range j.ResourceChanges {
		a := rc.Change.Actions
		if !a.NoOp() && !a.Read() && rc.Type != "local_file" {
			resources = append(resources, fmt.Sprintf("%s%s.%s.%s", changePrefix(rc.Change.Actions), rc.ModuleAddress, rc.Type, rc.Name))
		}
	}

	return strings.Join(resources, "\n")
}

func changePrefix(change tfjson.Actions) string {
	switch {
	case change.Create():
		return "+"
	case change.Update():
		return "~"
	case change.Delete():
		return "-"
	case change.DestroyBeforeCreate():
		return "-+"
	case change.CreateBeforeDestroy():
		return "+-"
	default:
		return "?"
	}
}

func (j *JSonPlanSummary) Name() string {
	return j.name
}

func (j *JSonPlanSummary) Changes() Changes {
	var create, update, deletes int
	for _, rc := range j.ResourceChanges {
		if rc.Type == "local_file" {
			// Local files are not interesting changes for our purposes
			continue
		}
		switch {
		case rc.Change.Actions.Create():
			create++
		case rc.Change.Actions.Delete():
			deletes++
		case rc.Change.Actions.Update():
			update++
		case rc.Change.Actions.CreateBeforeDestroy():
			create++
			deletes++
		case rc.Change.Actions.DestroyBeforeCreate():
			create++
			deletes++
		}
	}

	return Changes{Added: create, Updated: update, Deleted: deletes}
}

func (j *JSonPlanSummary) UpToDate() bool {
	return len(j.ResourceChanges) == 0
}

func ReadPlan(file string) (PlanSummary, error) {
	sum := &tfjson.Plan{}

	f, err := os.Open(file)

	if err != nil {
		return nil, err
	}

	defer f.Close()

	bytes, err := io.ReadAll(f)

	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(bytes, sum)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("while reading file %s:", file))
	}

	// Variables may be sensitive, so we don't want them.  They should not have been sent in the first place.
	sum.Variables = make(map[string]*tfjson.PlanVariable)

	return &JSonPlanSummary{
		Plan: sum,
		name: filepath.Base(file),
	}, nil
}
