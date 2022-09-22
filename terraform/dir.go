package terraform

import (
	"os"
	"path/filepath"
)

type PlanDir struct {
	name     string
	children []PlanSummary
}

func (p *PlanDir) Name() string {
	return p.name
}

func (p *PlanDir) Children() []PlanSummary {
	return p.children
}

func (p *PlanDir) Changes() Changes {
	var changes Changes

	for _, c := range p.children {
		c := c.Changes()
		changes.Added += c.Added
		changes.Updated += c.Updated
		changes.Deleted += c.Deleted
	}

	return changes.Summarize()
}

func (p *PlanDir) UpToDate() bool {
	for _, c := range p.children {
		if c.UpToDate() == false {
			return false
		}
	}

	return true
}

func ReadDir(dir string) (*PlanDir, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	result := &PlanDir{name: filepath.Base(dir)}

	for _, f := range files {
		var c PlanSummary
		var err error
		if f.IsDir() {
			c, err = ReadDir(filepath.Join(dir, f.Name()))
		} else {
			c, err = ReadPlan(filepath.Join(dir, f.Name()))
		}

		if err != nil {
			return nil, err
		}
		result.children = append(result.children, c)
	}

	return result, nil
}
