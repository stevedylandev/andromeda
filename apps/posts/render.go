package main

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/stevedylandev/andromeda/pkg/web"
)

func buildTemplates() (map[string]*template.Template, error) {
	pages, err := fs.Glob(appFS, "templates/*.html")
	if err != nil {
		return nil, err
	}

	out := make(map[string]*template.Template, len(pages))
	for _, page := range pages {
		name := path.Base(page)
		if name == "admin_base.html" {
			continue
		}

		// Public pages are templ components (see *.templ); what's left here is
		// the admin UI plus the standalone login page.
		patterns := []string{page}
		if strings.HasPrefix(name, "admin_") {
			patterns = append([]string{"templates/admin_base.html"}, patterns...)
		}

		tmpl, err := template.ParseFS(appFS, patterns...)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", page, err)
		}
		out[name] = tmpl
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
