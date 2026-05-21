package main

import (
	"net/http"

	"github.com/stevedylandev/andromeda/pkg/darkmatter"
	"github.com/stevedylandev/andromeda/pkg/web"
)

func (a *App) routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", a.indexHandler)
	mux.HandleFunc("POST /check", a.checkHandler)
	mux.HandleFunc("GET /static/", web.EmbeddedHandler(appFS, "static"))
	darkmatter.Mount(mux, "/assets")
	return mux
}
