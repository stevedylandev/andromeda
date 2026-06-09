package main

import (
	"net/http"
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
