package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path"
	"strings"
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
	a.renderPageStatus(w, http.StatusOK, name, data)
}

func (a *App) renderPageStatus(w http.ResponseWriter, status int, name string, data any) {
	tmpl, ok := a.Templates[name]
	if !ok {
		a.Log.Error("template missing", "name", name)
		http.Error(w, "template missing", http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		a.Log.Error("template render failed", "name", name, "err", err)
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}
