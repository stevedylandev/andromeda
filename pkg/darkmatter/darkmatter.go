// Package darkmatter ships the shared CSS + fonts used by andromeda Go apps.
package darkmatter

import (
	"embed"
	"mime"
	"net/http"
	"path"
	"strings"
)

//go:embed assets/darkmatter.css assets/index.html assets/fonts/*
var FS embed.FS

// Mount registers the darkmatter routes (css, fonts, gallery) on mux.
//   - <assetPrefix>/darkmatter.css
//   - <assetPrefix>/fonts/{file}
//   - /darkmatter and /darkmatter/ (gallery)
//
// assetPrefix is normally "/assets". The leading slash is required.
func Mount(mux *http.ServeMux, assetPrefix string) {
	assetPrefix = strings.TrimRight(assetPrefix, "/")
	if assetPrefix == "" {
		assetPrefix = "/assets"
	}

	mux.HandleFunc("GET "+assetPrefix+"/darkmatter.css", func(w http.ResponseWriter, r *http.Request) {
		serve(w, "assets/darkmatter.css", "text/css; charset=utf-8")
	})
	mux.HandleFunc("GET "+assetPrefix+"/fonts/{file}", func(w http.ResponseWriter, r *http.Request) {
		file := r.PathValue("file")
		ct := mime.TypeByExtension(path.Ext(file))
		if ct == "" {
			ct = "application/octet-stream"
		}
		serve(w, "assets/fonts/"+file, ct)
	})
	mux.HandleFunc("GET /darkmatter", func(w http.ResponseWriter, r *http.Request) {
		serve(w, "assets/index.html", "text/html; charset=utf-8")
	})
	mux.HandleFunc("GET /darkmatter/", func(w http.ResponseWriter, r *http.Request) {
		serve(w, "assets/index.html", "text/html; charset=utf-8")
	})
}

func serve(w http.ResponseWriter, name, contentType string) {
	data, err := FS.ReadFile(name)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", contentType)
	_, _ = w.Write(data)
}
