package main

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/stevedylandev/andromeda/crates-go/web"
)

func buildTemplates() (map[string]*template.Template, error) {
	pages, err := fs.Glob(appFS, "templates/*.html")
	if err != nil {
		return nil, err
	}

	out := make(map[string]*template.Template, len(pages))
	for _, page := range pages {
		if strings.HasSuffix(page, "/base.html") {
			continue
		}
		tmpl, err := template.ParseFS(appFS, "templates/base.html", page)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", page, err)
		}
		out[path.Base(page)] = tmpl
	}
	return out, nil
}

func (a *App) renderPage(w http.ResponseWriter, name string, data any) {
	tmpl, ok := a.Templates[name]
	if !ok {
		a.Log.Error("template missing", "name", name)
		http.Error(w, "template missing", http.StatusInternalServerError)
		return
	}
	web.Render(tmpl, w, name, data, a.Log)
}
