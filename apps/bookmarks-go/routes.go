package main

import (
	"net/http"

	"github.com/stevedylandev/andromeda/crates-go/auth"
	"github.com/stevedylandev/andromeda/crates-go/darkmatter"
	"github.com/stevedylandev/andromeda/crates-go/web"
)

func (a *App) routes() *http.ServeMux {
	mux := http.NewServeMux()

	requireSession := func(next http.HandlerFunc) http.HandlerFunc {
		return a.Sessions.RequireSession("/login", next)
	}
	requireAPIKey := func(next http.HandlerFunc) http.HandlerFunc {
		return auth.RequireAPIKey(a.APIKey, next)
	}

	mux.HandleFunc("GET /", a.indexHandler)
	mux.HandleFunc("GET /login", a.loginGetHandler)
	mux.HandleFunc("POST /login", a.loginPostHandler)
	mux.HandleFunc("GET /logout", a.logoutHandler)
	mux.HandleFunc("GET /admin", requireSession(a.adminHandler))
	mux.HandleFunc("POST /admin/categories", requireSession(a.adminAddCategory))
	mux.HandleFunc("POST /admin/categories/{short_id}/delete", requireSession(a.adminDeleteCategory))
	mux.HandleFunc("POST /admin/categories/{short_id}/move/{dir}", requireSession(a.adminMoveCategory))
	mux.HandleFunc("POST /admin/links", requireSession(a.adminAddLink))
	mux.HandleFunc("POST /admin/links/{short_id}/delete", requireSession(a.adminDeleteLink))

	mux.HandleFunc("GET /api/categories", a.apiListCategories)
	mux.HandleFunc("GET /api/links", a.apiListLinks)
	mux.HandleFunc("POST /api/links", requireAPIKey(a.apiCreateLink))

	mux.HandleFunc("GET /static/", web.EmbeddedHandler(appFS, "static"))
	darkmatter.Mount(mux, "/assets")
	return mux
}
