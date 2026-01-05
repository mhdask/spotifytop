package web

import (
	"embed"
	"net/http"
	"text/template"

	"github.com/rs/zerolog/log"
)

const (
	templatesExt = ".tmpl"
)

//go:embed templates/*
var templateFS embed.FS

func (w *Web) parseTemplate(name, path string) {
	if path == "" {
		path = name
	}

	if _, ok := w.Templates[name]; ok {
		log.Panic().Str("template", name).Msg("template already parsed once")
		return
	}

	w.Templates[name] = template.Must(template.ParseFS(templateFS, "templates/"+name+templatesExt, "templates/base"+templatesExt))
}

func (w *Web) templateGet(name string) *template.Template {
	if _, ok := w.Templates[name]; !ok {
		log.Error().Str("name", name).Msg("Trying to get a template that does not exists, returning a 404 page")
		return w.Templates["404.tmpl"]
	}

	return w.Templates[name]
}

func (w *Web) templateExec(rw http.ResponseWriter, r *http.Request, name string, data interface{}) {

	tmplData := struct {
		Errors []flashMessage
		Data   interface{}
	}{
		Errors: w.getFlash(rw, r),
		Data:   data,
	}

	if err := w.templateGet(name).ExecuteTemplate(rw, "base", tmplData); err != nil {
		log.Error().Err(err).Str("name", name).Interface("data", data).Msg("failed to view template")
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
}
