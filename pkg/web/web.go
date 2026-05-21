// Package web provides shared HTTP helpers used across andromeda Go apps.
package web

import (
	"embed"
	"encoding/json"
	"html/template"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
)

// EmbeddedHandler serves files from an embed.FS under the given URL prefix.
// Files are looked up relative to the same prefix inside the embedded FS.
func EmbeddedHandler(fs embed.FS, prefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/"+prefix+"/")
		path := filepath.ToSlash(filepath.Join(prefix, name))
		data, err := fs.ReadFile(path)
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

// Render executes a named template into w with status 200. Errors are logged
// and surfaced as HTTP 500.
func Render(t *template.Template, w http.ResponseWriter, name string, data any, log *slog.Logger) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, name, data); err != nil {
		if log != nil {
			log.Error("template render failed", "name", name, "err", err)
		}
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

// WriteJSON writes data as JSON with the given status code.
func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// DecodeJSON decodes r.Body into dst. On failure it writes a 400 JSON error
// and returns false.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid JSON")
		return false
	}
	return true
}

// WriteError writes a JSON error response of the form {"error": msg} with the
// given status code.
func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, map[string]any{"error": msg})
}

// PathInt64 parses a positive int64 path value from r. Returns (0, false) if
// missing, unparseable, or non-positive.
func PathInt64(r *http.Request, name string) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// RedirectWithError issues a 303 redirect to target with ?error=msg appended.
func RedirectWithError(w http.ResponseWriter, r *http.Request, target, msg string) {
	http.Redirect(w, r, target+"?error="+url.QueryEscape(msg), http.StatusSeeOther)
}

// RedirectWithSuccess issues a 303 redirect to target with ?success=msg appended.
func RedirectWithSuccess(w http.ResponseWriter, r *http.Request, target, msg string) {
	http.Redirect(w, r, target+"?success="+url.QueryEscape(msg), http.StatusSeeOther)
}
