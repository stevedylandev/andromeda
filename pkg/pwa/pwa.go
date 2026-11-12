// Package pwa provides the shared service worker, registration script, and
// offline fallback page used by andromeda apps so read-only pages keep
// working offline.
//
// Apps opt in by calling Mount in their route setup and adding
// <script src="/pwa.js" defer></script> to their templates. Their
// site.webmanifest should include start_url, scope, and id.
package pwa

import (
	"embed"
	"net/http"
)

//go:embed static/sw.js static/pwa.js static/offline.html
var files embed.FS

// Mount registers the PWA routes on mux:
//   - GET /sw.js    the service worker (served from the root so its default
//     scope covers the whole origin)
//   - GET /pwa.js   the registration script included by app templates
//   - GET /offline  fallback page shown when an uncached navigation fails
func Mount(mux *http.ServeMux) {
	mux.HandleFunc("GET /sw.js", func(w http.ResponseWriter, r *http.Request) {
		serve(w, "static/sw.js", "application/javascript; charset=utf-8", true)
	})
	mux.HandleFunc("GET /pwa.js", func(w http.ResponseWriter, r *http.Request) {
		serve(w, "static/pwa.js", "application/javascript; charset=utf-8", true)
	})
	mux.HandleFunc("GET /offline", func(w http.ResponseWriter, r *http.Request) {
		serve(w, "static/offline.html", "text/html; charset=utf-8", false)
	})
}

func serve(w http.ResponseWriter, name, contentType string, noCache bool) {
	data, err := files.ReadFile(name)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", contentType)
	if noCache {
		// Browsers cache service worker scripts for up to 24h by default.
		// Always revalidate so worker updates ship with each binary.
		w.Header().Set("Cache-Control", "no-cache")
	}
	_, _ = w.Write(data)
}
