package storage

import (
	"encoding/json"
	"fmt"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/deweysasser/olympus/git"
	"github.com/deweysasser/olympus/run"
	"github.com/deweysasser/olympus/terraform"
	"github.com/pkg/errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type Storage struct {
	dir        string
	branches   mapset.Set[git.Branch]
	workspaces mapset.Set[terraform.Workspace]
}

type Key []string

func ParseKey(s string) Key {
	return Key(strings.Split(s, "/"))
}

const timeFormat = "2006-01-02-15-04-05"

func (s *Storage) buildFile(key Key, r *run.PlanRecord) string {
	return filepath.Join(
		s.dir,
		filepath.Join(
			key...,
		),
		fmt.Sprintf("%s__%s__%s.json", r.End.Format(timeFormat), r.Branch, r.Workspace),
	)
}

func (s *Storage) Store(key Key, r *run.PlanRecord) error {
	file := s.buildFile(key, r)
	if bytes, err := json.Marshal(r); err != nil {
		return err
	} else {
		dir := filepath.Dir(file)
		dirinfo, err := os.Stat(dir)
		switch {
		case err != nil:
			os.MkdirAll(dir, os.ModePerm)
		case !dirinfo.IsDir():
			return errors.New("Path " + dir + " is exists but is not a directory")
		}
		err = os.WriteFile(file, bytes, os.ModePerm)
		if err == nil {
			s.branches.Add(r.Branch)
			s.workspaces.Add(r.Workspace)
		}
		return err
	}
}

func (s *Storage) Branches() mapset.Set[git.Branch] {
	return s.branches
}
func (s *Storage) Workspaces() mapset.Set[terraform.Workspace] {
	return s.workspaces
}

func (s *Storage) Summary(key Key, branch git.Branch, workspace terraform.Workspace) (run.Set, error) {
	return run.Set{}, errors.New("Not yet implemented")
}

func New(dir string) *Storage {
	s := &Storage{
		dir:        dir,
		branches:   mapset.NewSet[git.Branch](),
		workspaces: mapset.NewSet[terraform.Workspace](),
	}

	s.readFileNamesForMetadata()
	return s
}

func (s *Storage) readFileNamesForMetadata() {
	filepath.Walk(s.dir, func(path string, info fs.FileInfo, err error) error {
		if info != nil && !info.IsDir() {
			if parts := strings.Split(info.Name(), "__"); len(parts) > 2 {
				s.branches.Add(git.Branch(parts[1]))
				lastPart := parts[2]
				ext := filepath.Ext(lastPart)
				s.workspaces.Add(terraform.Workspace(lastPart[:len(lastPart)-len(ext)]))
			}
		}
		return nil
	})
}
