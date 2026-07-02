package main

import (
	"net/http"

	"github.com/stevedylandev/andromeda/pkg/darkmatter"
	"github.com/stevedylandev/andromeda/pkg/web"
)

func (a *App) routes() *http.ServeMux {
	mux := http.NewServeMux()

	requireSession := func(next http.HandlerFunc) http.HandlerFunc {
		return a.Sessions.RequireSession("/admin/login", next)
	}

	mux.HandleFunc("GET /", a.indexHandler)
	mux.HandleFunc("GET /static/", web.EmbeddedHandler(appFS, "static"))
	darkmatter.Mount(mux, "/assets")

	mux.HandleFunc("GET /admin/login", a.loginGetHandler)
	mux.HandleFunc("POST /admin/login", a.loginPostHandler)
	mux.HandleFunc("GET /admin/logout", a.logoutHandler)
	mux.HandleFunc("GET /admin", requireSession(a.adminHandler))
	mux.HandleFunc("POST /admin/add", requireSession(a.adminAddQuote))
	mux.HandleFunc("POST /admin/quotes/{short_id}/delete", requireSession(a.adminDeleteQuote))

	mux.HandleFunc("GET /api/quotes", a.apiListQuotes)
	mux.HandleFunc("GET /api/quotes/today", a.apiQuoteOfTheDay)
	mux.HandleFunc("GET /api/quotes/{short_id}", a.apiGetQuote)

	return mux
}
