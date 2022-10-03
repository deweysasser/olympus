package terraform

import (
	"github.com/floatdrop/lru"
	"github.com/rs/zerolog/log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
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
		changes.ResourcesAdded += c.ResourcesAdded
		changes.ResourcesUpdated += c.ResourcesUpdated
		changes.ResourcesDeleted += c.ResourcesDeleted
	}

	return changes
}

func (p *PlanDir) ChangedResources() string {
	var resources []string

	for _, rc := range p.children {
		resources = append(resources, rc.ChangedResources())
	}

	return strings.Join(resources, "\n")
}

func (p *PlanDir) UpToDate() bool {
	for _, c := range p.children {
		if !c.UpToDate() {
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
		go func(dir string, f os.DirEntry) {
			defer wg.Done()
			c, err := readFile(dir, f)
			if err == nil {
				children <- c
			}
		}(dir, f)
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

type cacheEntry struct {
	plan     PlanSummary
	fileTime time.Time
}

var cache = lru.New[string, cacheEntry](50)

func readFile(dir string, f os.DirEntry) (PlanSummary, error) {
	path := filepath.Join(dir, f.Name())
	info, err := f.Info()
	var c PlanSummary

	if err != nil {
		return c, err
	}

	cached := cache.Get(path)
	if cached != nil && cached.fileTime == info.ModTime() {
		return cached.plan, nil
	}

	if f.IsDir() {
		c, err = ReadDir(path)
	} else {
		c, err = ReadPlan(path)

		if err == nil {
			cache.Set(path, cacheEntry{
				plan:     c,
				fileTime: info.ModTime(),
			})
		}
	}

	if err != nil {
		log.Error().Err(err).Msg("Error reading plan")
		return nil, err
	}
	return c, nil
}
