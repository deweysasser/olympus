package terraform

import (
	"github.com/rs/zerolog/log"
	"os"
	"path/filepath"
	"sync"
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

	wg := sync.WaitGroup{}
	children := make(chan PlanSummary)

	for _, f := range files {
		wg.Add(1)
		go func(f os.DirEntry) {
			defer wg.Done()
			var c PlanSummary
			var err error
			if f.IsDir() {
				c, err = ReadDir(filepath.Join(dir, f.Name()))
			} else {
				c, err = ReadPlan(filepath.Join(dir, f.Name()))
			}

			if err != nil {
				log.Error().Err(err).Msg("Error reading plan")
				return
			}
			children <- c
		}(f)
	}
	go func() {
		defer close(children)
		defer wg.Wait()
	}()

	for child := range children {
		result.children = append(result.children, child)
	}

	return result, nil
}
