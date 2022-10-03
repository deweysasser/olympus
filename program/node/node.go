package node

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/deweysasser/olympus/git"
	"github.com/deweysasser/olympus/run"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/remeh/sizedwaitgroup"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type Options struct {
	Collector   string    `help:"collector address" default:"http://localhost:8081"`
	Terraform   Terraform `embed:"" prefix:"terraform."`
	Parallel    int       `help:"Number of processes to run in parallel" default:"1"`
	Directories []string  `arg:"" help:"Directories in which to run terraform"`
	ClipLast    int       `help:"Number of directories from the end path to use sending to poc-server"`
}

type Terraform struct {
	PlanCommand     string        `aliases:"plan" help:"Command to run in the terraform directory to produce the plan" default:"terraform plan -o plan.bin"`
	ShowPlanCommand string        `aliases:"show-plan" help:"Command to run to output the plan in terraform json" default:"terraform show plan --json plan.bin"`
	PlanFile        string        `aliases:"file" help:"Name of the created plan file" default:"plan.bin"`
	RunTimeout      time.Duration `help:"Maximum time to allow a command to run" default:"5m"`
}

func (options *Options) Run() error {

	log.Debug().Int("parallel", options.Parallel).Msg("Running plans concurrently")
	wg := sizedwaitgroup.New(options.Parallel)

	durations := make(chan time.Duration, 10000)

	start := time.Now()

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
			start := time.Now()
			options.processDir(dir)
			durations <- time.Since(start)
		}(dir)
	}

	wg.Wait()
	close(durations)

	fmt.Print("Durations:")
	var total time.Duration
	var count int64
	for d := range durations {
		total = total + d
		count++
		fmt.Print(" ", d.String())
	}

	average := total.Milliseconds() / count
	aDur := time.Duration(average) * time.Millisecond

	fmt.Println("")
	fmt.Println("Total duration", time.Since(start).String())
	fmt.Println("Average duration", aDur.String())
	fmt.Println("planes ", count)

	return nil
}

// processDir processes a single directory
func (options *Options) processDir(dir string) {

	log := log.Logger.With().Str("dir", dir).Logger()
	log.Info().Msg("Processing dir")
	sha, err := git.CurrentSHA(dir)

	if err != nil {
		log.Error().Err(err).Msg("Failed to get git HEAD sha")
	}

	run := &run.PlanRecord{Start: time.Now(), CommitSHA: sha}

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
	log.Info().Str("url", url).Msg("Posting results")
	_, err = http.Post(url, "text/json", bytes.NewReader(b))
	if err != nil {
		log.Error().Err(err).Msg("Failed to send results")
	}
}

func (options *Options) getPlan(dir string) (*tfjson.Plan, error) {
	cmd := strings.Split(options.Terraform.PlanCommand, " ")
	ctx, cancel := context.WithTimeout(context.Background(), options.Terraform.RunTimeout+60*time.Second)
	defer cancel()

	command := exec.CommandContext(ctx, cmd[0], cmd[1:]...)
	command.Dir = dir

	log := log.Logger.With().Str("dir", dir).Logger()

	var err error

	clog := log.With().Str("command", strings.Join(cmd, " ")).Logger()

	clog.Debug().Msg("running command")

	done := sigintAfter(options, command)
	err = command.Run()
	done()
	if err != nil {
		log.Error().Err(err).Msg("Error running command")
		return nil, err
	}

	cmd = strings.Split(options.Terraform.ShowPlanCommand, " ")
	ctx, cancel = context.WithTimeout(context.Background(), options.Terraform.RunTimeout)
	defer cancel()

	command = exec.CommandContext(ctx, cmd[0], cmd[1:]...)
	command.Dir = dir

	clog = log.With().Str("command", strings.Join(cmd, " ")).Logger()

	clog.Debug().Msg("running command")
	done = sigintAfter(options, command)
	bytes, err := command.Output()
	done()

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

func sigintAfter(options *Options, command *exec.Cmd) func() {
	done := make(chan interface{})
	go func() {
		select {
		case <-done:
			return
		case <-time.After(options.Terraform.RunTimeout):
			log.Debug().Str("timeout", options.Terraform.RunTimeout.String()).Msg("command exceeded run time.  Sending interrupt")
			command.Process.Signal(syscall.SIGINT)
		}
	}()

	return func() {
		close(done)
	}
}
