package ui

import (
	"embed"
	"fmt"
	"github.com/deweysasser/olympus/middleware"
	"github.com/deweysasser/olympus/terraform"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

//go:embed files
var files embed.FS

type Options struct {
	Port            int           `help:"Port on which to listen" default:"8080"`
	TemplateReloads time.Duration `help:"frequency at which to reload templates" default:"500ms"`
	UIFilePath      string        `help:"path for HTML templates" type:"path" optional:"1"`
	DataPath        string        `help:"Path to find data" type:"path" default:"received"`

	templates *template.Template
	Meta      SiteMeta `embed:"" prefix:"site."`
}

type SiteMeta struct {
	Name string `help:"Site name for display on tab" default:"Olympus"`
}

func (ui *Options) Run() error {

	server, err := ui.Router()
	if err != nil {
		return err
	}

	server.Use(middleware.RequestLogger)

	log.Debug().Int("port", ui.Port).Msg("Server listening")

	return http.ListenAndServe(fmt.Sprintf(":%d", ui.Port), server)
}

func (ui *Options) Router() (*mux.Router, error) {
	server := mux.NewRouter()

	d, err := os.Stat(ui.DataPath)

	if err == nil {
		if !d.IsDir() {
			return nil, errors.New("Data directory not a directory: " + ui.DataPath)
		}
	} else if os.IsNotExist(err) {
		log.Debug().Str("dir", ui.DataPath).Msg("Creating dir")
		if e := os.MkdirAll(ui.DataPath, os.ModePerm); e != nil {
			return nil, e
		}
	}

	ui.templates, err = ui.parseTemplates()

	if err != nil {
		return nil, err
	}

	if ui.UIFilePath != "" {
		go func() {
			log.Debug().Str("every", ui.TemplateReloads.String()).Msg("Reloading templates periodically")
			for range time.Tick(ui.TemplateReloads) {
				if t, err := ui.parseTemplates(); err != nil {
					log.Error().Err(err).Msg("Failed to reload templates")
				} else {
					// log.Debug().Msg("templates reloaded")
					ui.templates = t
				}
			}
		}()
	}

	server.Path("/status").HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte("alive"))
	})

	fs, err := ui.ServeStatic()

	if err != nil {
		return nil, err
	}

	server.PathPrefix("/static").Handler(fs)
	server.PathPrefix("/").HandlerFunc(ui.Render)

	return server, nil
}

func (ui *Options) parseTemplates() (*template.Template, error) {
	if ui.UIFilePath != "" {
		templates := filepath.Join(ui.UIFilePath, "templates")

		info, err := os.Stat(templates)
		if err != nil {
			return nil, errors.New("Failed to find template dir:" + templates)
		}

		if !info.IsDir() {
			return nil, errors.New("Path must be a directory:" + templates)
		}

		t, err := template.ParseGlob(templates + "/*.html")
		if err != nil {
			return nil, err
		}
		return t, err

	} else {
		sub, err := fs.Sub(files, "files/templates")
		if err != nil {
			return nil, err
		}
		log.Debug().Msg("Loading templates from embed")
		t, err := template.ParseFS(sub, "*.html")
		if err != nil {
			return nil, err
		}
		return t, err
	}
}

func (ui *Options) ServeStatic() (http.Handler, error) {

	if ui.UIFilePath != "" {
		staticFiles := filepath.Join(ui.UIFilePath, "static")

		info, err := os.Stat(staticFiles)
		if err != nil {
			return nil, errors.New("Failed to find static files dir:" + staticFiles)
		}

		if !info.IsDir() {
			return nil, errors.New("Path must be a directory:" + staticFiles)
		}

		return http.FileServer(http.Dir(staticFiles)), nil
	} else {
		sub, err := fs.Sub(files, "files")
		if err != nil {
			return nil, err
		}
		return http.FileServer(http.FS(sub)), nil
	}
}

func (ui *Options) Render(writer http.ResponseWriter, request *http.Request) {
	log := log.Logger.With().Str("uri", request.RequestURI).Logger()

	dir := ui.DataPath
	if request.RequestURI != "/" {
		dir = filepath.Join(dir, request.RequestURI[1:])
	}

	log.Debug().Str("data_dir", dir).Msg("Reading plan data")

	summaries, err := terraform.ReadDir(dir)
	if err != nil {
		log.Debug().Err(err).Str("dir", dir).Msg("could not read data")
		http.NotFound(writer, request)
		return
	}

	data := map[string]any{
		"site": ui.Meta,
		"data": CreateTable(summaries.Children()),
	}

	err = ui.templates.ExecuteTemplate(writer, "index.html", data)
	if err != nil {
		log.Error().Err(err).Msg("Error evaluating template")
	}
}
