package poc_server

import (
	"encoding/json"
	"fmt"
	"github.com/deweysasser/olympus/middleware"
	"github.com/deweysasser/olympus/run"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
)

type Options struct {
	Port          int    `help:"Port on which to listen" default:"8081"`
	DataDirectory string `help:"Directory into which to write data" type:"existingdir" default:"received"`
	fs            fs.FS  // file system to use for operations
}

func (o *Options) Run() error {
	server := o.createServer()

	log.Debug().Int("port", o.Port).Msg("Listening")
	return http.ListenAndServe(fmt.Sprintf(":%d", o.Port), server)
}

func (o *Options) createServer() *mux.Router {
	server := mux.NewRouter()

	server.Use(middleware.RequestLogger)

	server.Path("/status").HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		alive := map[string]string{
			"status": "alive",
		}
		bytes, err := json.Marshal(alive)
		if err != nil {
			writer.WriteHeader(500)
		} else {
			writer.Write(bytes)
		}
	})

	server.PathPrefix("/").Methods("POST").HandlerFunc(o.receive)

	return server
}

func (o *Options) receive(writer http.ResponseWriter, request *http.Request) {
	path := filepath.Join(o.DataDirectory, request.RequestURI[1:])
	log := log.With().Str("path", path).Logger()

	info, err := os.Stat(path)
	switch {
	case err != nil:
		err = os.MkdirAll(path, os.ModePerm)
		if err != nil {
			log.Error().Err(err).Msg("Failed to make directories")
		}
	case !info.IsDir():
		log.Error().Err(err).Msg("output directory isn't")
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	// TODO:  save the whole thing, not just the plan

	outputPath := filepath.Join(path, "plan.json")

	log = log.With().Str("output_file", outputPath).Logger()

	f, err := os.Create(outputPath)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create output file")
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer f.Close()

	run := &run.PlanRecord{}
	bytes, err := io.ReadAll(request.Body)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read request")
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = json.Unmarshal(bytes, run)
	if err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal request")
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	bytes, err = json.Marshal(run.Plan)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal output")
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Debug().Msg("Wrote output file")

	f.Write(bytes)
}
