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
	mux.HandleFunc("GET /admin/search", requireSession(a.adminSearch))
	mux.HandleFunc("POST /admin/categories/labels", requireSession(a.adminUpdateLabels))
	mux.HandleFunc("POST /admin/add", requireSession(a.adminAddBook))
	mux.HandleFunc("POST /admin/books/{id}/status", requireSession(a.adminUpdateStatus))
	mux.HandleFunc("POST /admin/books/{id}/notes", requireSession(a.adminUpdateNotes))
	mux.HandleFunc("POST /admin/books/{id}/delete", requireSession(a.adminDeleteBook))

	mux.HandleFunc("GET /api/books", a.apiListBooks)
	mux.HandleFunc("GET /api/books/{id}", a.apiGetBook)

	return mux
}
