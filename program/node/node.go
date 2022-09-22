package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/deweysasser/olympus/git"
	"github.com/deweysasser/olympus/program/server"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/remeh/sizedwaitgroup"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Options struct {
	Collector   string    `help:"collector address" default:"http://localhost:8081"`
	Terraform   Terraform `embed:"" prefix:"terraform."`
	Parallel    int       `help:"Number of processes to run in parallel" default:"1"`
	Directories []string  `arg:"" help:"Directories in which to run terraform"`
	ClipLast    int       `help:"Number of directories from the end path to use sending to server"`
}

type Terraform struct {
	PlanCommand     string `aliases:"plan" help:"Command to run in the terraform directory to produce the plan" default:"terraform plan -o plan.bin"`
	ShowPlanCommand string `aliases:"show-plan" help:"Command to run to output the plan in terraform json" default:"terraform show plan --json plan.bin"`
	PlanFile        string `aliases:"file" help:"Name of the created plan file" default:"plan.bin"`
}

func (options *Options) Run() error {

	log.Debug().Int("parallel", options.Parallel).Msg("Running plans concurrently")
	wg := sizedwaitgroup.New(options.Parallel)

	for _, dir := range options.Directories {
		wg.Add()
		go func(dir string) {
			defer wg.Done()
			info, err := os.Stat(dir)
			if err != nil {
				log.Error().Err(err).Str("dir", dir).Msg("Directory not found")
				return
			} else if !info.IsDir() {
				log.Info().Str("dir", dir).Msg("Directory is not a directory.  Skipping")
				return
			}
			options.processDir(dir)
		}(dir)
	}

	wg.Wait()

	return nil
}

// processDir processes a single directory
func (options *Options) processDir(dir string) {

	log := log.Logger.With().Str("dir", dir).Logger()
	log.Debug().Msg("Processing dir")
	sha, err := git.CurrentSHA(dir)

	if err != nil {
		log.Error().Err(err).Msg("Failed to get git HEAD sha")
	}

	run := &server.PlanRecord{Start: time.Now(), Hash: sha}

	plan, err := options.getPlan(dir)
	if err != nil {
		return
	}

	run.Plan = plan

	b, err := json.Marshal(run)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get git HEAD sha")
		return
	}

	if options.ClipLast > 0 {
		parts := strings.Split(dir, "/") // todo:  better separator
		if len(parts) > options.ClipLast {
			last := len(parts) - options.ClipLast
			parts = parts[last:]

			dir = filepath.Join(parts...)
		}
	}

	url := fmt.Sprintf("%s/%s", options.Collector, dir)
	log.Debug().Str("url", url).Msg("Posting results")
	_, err = http.Post(url, "text/json", bytes.NewReader(b))
	if err != nil {
		log.Error().Err(err).Msg("Failed to send results")
	}
}

func (options *Options) getPlan(dir string) (*tfjson.Plan, error) {
	cmd := strings.Split(options.Terraform.PlanCommand, " ")
	command := exec.Command(cmd[0], cmd[1:]...)
	command.Dir = dir

	log := log.Logger.With().Str("dir", dir).Logger()

	var err error

	clog := log.With().Str("command", strings.Join(cmd, " ")).Logger()

	clog.Debug().Msg("running command")
	err = command.Run()
	if err != nil {
		log.Error().Err(err).Msg("Error running command")
		return nil, err
	}

	cmd = strings.Split(options.Terraform.ShowPlanCommand, " ")
	command = exec.Command(cmd[0], cmd[1:]...)
	command.Dir = dir

	clog = log.With().Str("command", strings.Join(cmd, " ")).Logger()

	clog.Debug().Msg("running command")
	bytes, err := command.Output()

	if err != nil {
		clog.Error().Err(err).Msg("Error running command")
		return nil, err
	}

	var plan tfjson.Plan
	err = json.Unmarshal(bytes, &plan)
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse json output")
	}

	// Get rid of variables immediately -- they likely contain sensitive information
	plan.Variables = make(map[string]*tfjson.PlanVariable)

	return &plan, nil
}