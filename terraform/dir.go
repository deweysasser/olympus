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

func (J *PlanDir) Children() []PlanSummary {
	return J.children
}

func (p *PlanDir) Changes() (int, int, int) {
	var toadd, toupdate, todelete int

	for _, c := range p.children {
		a, u, d := c.Changes()
		toadd += a
		toupdate += u
		todelete += d
	}

	return toadd, toupdate, todelete
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
