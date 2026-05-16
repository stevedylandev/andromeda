package main

import (
	"encoding/json"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

func (a *App) embeddedHandler(prefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/"+prefix+"/")
		path := filepath.ToSlash(filepath.Join(prefix, name))
		data, err := appFS.ReadFile(path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if ct := mime.TypeByExtension(filepath.Ext(path)); ct != "" {
			w.Header().Set("Content-Type", ct)
		}
		_, _ = w.Write(data)
	}
}

func (a *App) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.Templates.ExecuteTemplate(w, name, data); err != nil {
		a.Log.Error("template render failed", "name", name, "err", err)
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON"})
		return false
	}
	return true
}
