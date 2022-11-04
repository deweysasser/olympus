package run

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/acarl005/stripansi"
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
	Collector  string        `help:"collector address" default:"http://localhost:8080/plan"`
	Command    []string      `sep:";" help:"sequences of commands to generate a plan JSON.  The final command should generate a terraform JSON format plan output" default:"terraform plan; terraform show -json plan"`
	RunTimeout time.Duration `help:"Maximum time to allow a command to run" default:"5m"`
	Parallel   int           `help:"Number of processes to run in parallel" default:"1"`
	ClipLast   int           `help:"Number of directories from the end path to use sending to poc-server" default:"2"`

	Directories []string `arg:"" help:"Directories in which to run terraform"`
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
		total += d
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

	var env []string

	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "TERM=") {
			env = append(env, e)
		}
	}

	for _, c := range options.Command[0 : len(options.Command)-1] {

		cmd := strings.Split(strings.TrimSpace(c), " ")
		ctx, cancel := context.WithTimeout(context.Background(), options.RunTimeout+60*time.Second)
		defer cancel()

		command := exec.CommandContext(ctx, cmd[0], cmd[1:]...)
		command.Dir = dir
		command.Env = env

		log := log.Logger.With().Str("dir", dir).Logger()

		var err error

		clog := log.With().Str("command", strings.Join(cmd, " ")).Logger()

		clog.Debug().Msg("running command")

		done := sigintAfter(options, command)
		bytes, err := command.CombinedOutput()
		done()
		if err != nil {
			log.Error().Err(err).Str("command", command.String()).Str("output", stripansi.Strip(string(bytes))).Msg("Error running command")

			return nil, err
		}
	}

	cmd := strings.Split(strings.TrimSpace(options.Command[len(options.Command)-1]), " ")
	ctx, cancel := context.WithTimeout(context.Background(), options.RunTimeout)
	defer cancel()

	command := exec.CommandContext(ctx, cmd[0], cmd[1:]...)
	command.Dir = dir
	command.Env = env

	clog := log.With().Str("command", strings.Join(cmd, " ")).Logger()

	clog.Debug().Msg("running command")
	done := sigintAfter(options, command)
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
		case <-time.After(options.RunTimeout):
			log.Debug().Str("timeout", options.RunTimeout.String()).Msg("command exceeded run time.  Sending interrupt")
			command.Process.Signal(syscall.SIGINT)
		}
	}()

	return func() {
		close(done)
	}
}
