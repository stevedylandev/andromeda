package main

import (
	"mime"
	"net/http"
	"path"
)

func (a *App) routes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /assets/kepler.css", func(w http.ResponseWriter, r *http.Request) {
		data, err := appFS.ReadFile("static/styles.css")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		_, _ = w.Write(data)
	})
	mux.HandleFunc("GET /assets/fonts/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		data, err := appFS.ReadFile("static/fonts/" + name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "font/otf")
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		_, _ = w.Write(data)
	})
	// Static icon/manifest assets. Registered as explicit literal paths so
	// they stay more specific than the "/{repo}/..." routes (a wildcard
	// "/assets/{name}" would be ambiguous against "/{repo}/refs" etc.).
	staticAsset := func(w http.ResponseWriter, r *http.Request) {
		name := path.Base(r.URL.Path)
		data, err := appFS.ReadFile("static/" + name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		ct := mime.TypeByExtension(path.Ext(name))
		if ct == "" {
			ct = http.DetectContentType(data)
		}
		w.Header().Set("Content-Type", ct)
		w.Header().Set("Cache-Control", "public, max-age=86400")
		_, _ = w.Write(data)
	}
	for _, name := range []string{
		"og.png", "favicon.ico", "favicon-16x16.png", "favicon-32x32.png",
		"apple-touch-icon.png", "android-chrome-192x192.png",
		"android-chrome-512x512.png", "site.webmanifest",
	} {
		mux.HandleFunc("GET /assets/"+name, staticAsset)
	}

	mux.HandleFunc("GET /api/repos", a.apiReposHandler)

	mux.HandleFunc("GET /{$}", a.indexHandler)
	mux.HandleFunc("GET /{repo}", a.repoHandler)
	mux.HandleFunc("GET /{repo}/refs", a.refsHandler)
	mux.HandleFunc("GET /{repo}/atom.xml", a.atomHandler)
	mux.HandleFunc("GET /{repo}/log/{ref}", a.logHandler)
	mux.HandleFunc("GET /{repo}/commit/{sha}", a.commitHandler)
	mux.HandleFunc("GET /{repo}/tree/{ref}", a.treeHandler)
	mux.HandleFunc("GET /{repo}/tree/{ref}/{path...}", a.treeHandler)
	mux.HandleFunc("GET /{repo}/blob/{ref}/{path...}", a.blobHandler)
	mux.HandleFunc("GET /{repo}/raw/{ref}/{path...}", a.rawHandler)
	mux.HandleFunc("GET /{repo}/archive/{name}", a.archiveHandler)

	return mux
}
