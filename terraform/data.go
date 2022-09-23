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
	// Changes calculates the nubmer of additions, changes, and deletions
	Changes() Changes
	UpToDate() bool
	Children() []PlanSummary
	ChangedResources() string
}

type Changes struct {
	Added, Updated, Deleted int
	HasAny                  bool
	Highest                 string
}

func (c Changes) Summarize() Changes {
	c.HasAny = c.Added+c.Updated+c.Deleted > 0
	if c.Deleted > 0 {
		c.Highest = "deleted"
	} else if c.Updated > 0 {
		c.Highest = "updated"
	} else if c.Added > 0 {
		c.Highest = "added"
	} else {
		c.Highest = "none"
	}
	return c
}

func (J *JSonPlanSummary) Children() []PlanSummary {
	return []PlanSummary{}
}

type JSonPlanSummary struct {
	*tfjson.Plan
	name string
}

func (J *JSonPlanSummary) ChangedResources() string {
	var resources []string

	for _, rc := range J.ResourceChanges {
		a := rc.Change.Actions
		if !a.NoOp() && !a.Read() {
			resources = append(resources, fmt.Sprintf("%s%s.%s", changePrefix(rc.Change.Actions), rc.Type, rc.Name))
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

func (J *JSonPlanSummary) Name() string {
	return J.name
}

func (J *JSonPlanSummary) Changes() Changes {
	var create, update, deletes int
	for _, rc := range J.ResourceChanges {
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

	return Changes{Added: create, Updated: update, Deleted: deletes}.Summarize()
}

func (J *JSonPlanSummary) UpToDate() bool {
	return len(J.ResourceChanges) == 0
}

func ReadPlan(file string) (*JSonPlanSummary, error) {
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
