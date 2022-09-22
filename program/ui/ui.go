package ui

import (
	"fmt"
	"github.com/deweysasser/golang-program/terraform"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Options struct {
	Port            int           `help:"Port on which to listen" default:"8080"`
	TemplateReloads time.Duration `help:"frequency at which to reload templates" default:"500ms"`
	TemplatePath    string        `help:"path for HTML templates" type:"existingdir" default:"ui"`

	templates *template.Template
	Meta      SiteMeta `embed:"" prefix:"site."`
}

type SiteMeta struct {
	Name string `help:"Site name for display on tab" default:"Olympus"`
}

func (ui *Options) Run() error {
	server := mux.NewRouter()

	server.Use(loggingMiddleware)

	var err error
	ui.templates, err = ui.parseTemplates()

	go func() {
		log.Debug().Str("every", ui.TemplateReloads.String()).Msg("Reloading templates periodically")
		for range time.Tick(ui.TemplateReloads) {
			if t, err := ui.parseTemplates(); err != nil {
				log.Error().Err(err).Msg("Failed to reload templates")
			} else {
				//log.Debug().Msg("templates reloaded")
				ui.templates = t
			}
		}
	}()

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to parse templates")
		return err
	}

	server.PathPrefix("/static").HandlerFunc(ui.ServeStatic)
	server.PathPrefix("/").HandlerFunc(ui.Render)

	log.Debug().Int("port", ui.Port).Msg("Server listening")
	return http.ListenAndServe(fmt.Sprintf(":%d", ui.Port), server)
}

func (ui *Options) parseTemplates() (*template.Template, error) {
	var err error
	t, err := template.ParseGlob(ui.uiFilePath("*.html"))
	return t, err
}

func (ui *Options) uiFilePath(path string) string {
	return fmt.Sprintf("%s/%s", ui.TemplatePath, path)
}

func (ui *Options) ServeStatic(writer http.ResponseWriter, request *http.Request) {
	log := log.Logger.With().Str("uri", request.RequestURI).Logger()

	path := ui.uiFilePath(request.RequestURI[1:])

	log = log.With().Str("path", path).Logger()

	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		log.Debug().Msg("Serving static file")
		if serveStaticFile(writer, request, path, log) {
			return
		}
	} else {
		http.NotFound(writer, request)
	}

}

func (ui *Options) Render(writer http.ResponseWriter, request *http.Request) {
	log := log.Logger.With().Str("uri", request.RequestURI).Logger()

	dir := "data"
	if request.RequestURI != "/" {
		dir = filepath.Join(dir, request.RequestURI[1:])
	}

	log.Debug().Str("data_dir", dir).Msg("Reading plan data")

	summaries, err := terraform.ReadDir(dir)
	if err != nil {
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

	return

}

func mimetype(file string) string {
	i := strings.LastIndex(strings.ToLower(file), ".")
	if i < 1 {
		return "text/plain"
	}

	s := file[i+1:]
	log.Debug().Str("path", file).Str("ext", s).Msg("Finding mimetype")
	switch s {
	case "jpg":
		return "image/jpg"
	case "png":
		return "image/png"
	case "css":
		return "text/css"
	default:
		return "text/plain"
	}
}

func serveStaticFile(writer http.ResponseWriter, request *http.Request, path string, log zerolog.Logger) bool {
	writer.Header().Add("Content-Type", mimetype(path))
	p := make([]byte, 2048)
	f, err := os.Open(path)
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		http.NotFound(writer, request)
		return true
	}
	defer f.Close()

	for {
		n, err := f.Read(p)
		switch {
		case err == io.EOF:
			return true
		case n == 0:
			return true
		case err == nil:
			writer.Write(p[:n])
		default:
			log.Error().Err(err).Msg("Error reading file during response")
		}
	}
	return false
}
