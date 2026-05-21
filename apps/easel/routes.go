package main

import (
	"net/http"

	"github.com/stevedylandev/andromeda/pkg/darkmatter"
	"github.com/stevedylandev/andromeda/pkg/web"
)

func (a *App) routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", a.indexHandler)
	mux.HandleFunc("GET /day/{date}", a.dayHandler)
	mux.HandleFunc("GET /archive", a.archiveHandler)
	mux.HandleFunc("GET /feed.xml", a.withCORS(a.atomFeedHandler))
	mux.HandleFunc("GET /api/today", a.withCORS(a.apiToday))
	mux.HandleFunc("GET /api/day/{date}", a.withCORS(a.apiDay))
	mux.HandleFunc("GET /api/archive", a.withCORS(a.apiArchive))
	mux.HandleFunc("GET /static/", web.EmbeddedHandler(appFS, "static"))
	darkmatter.Mount(mux, "/assets")
	return mux
}

func (a *App) withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET")
		next(w, r)
	}
}
