package main

import (
	"net/http"

	"github.com/stevedylandev/andromeda/crates-go/auth"
	"github.com/stevedylandev/andromeda/crates-go/darkmatter"
	"github.com/stevedylandev/andromeda/crates-go/web"
)

func (a *App) routes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /static/", web.EmbeddedHandler(appFS, "static"))
	darkmatter.Mount(mux, "/assets")

	requireSession := func(next http.HandlerFunc) http.HandlerFunc {
		return a.Sessions.RequireSession("/login", next)
	}
	requireAPIKey := func(next http.HandlerFunc) http.HandlerFunc {
		return auth.RequireAPIKey(a.APIKey, next)
	}

	mux.HandleFunc("GET /login", a.loginGetHandler)
	mux.HandleFunc("POST /login", a.loginPostHandler)
	mux.HandleFunc("GET /logout", a.logoutHandler)

	mux.HandleFunc("GET /{$}", requireSession(a.indexHandler))
	mux.HandleFunc("GET /notes/new", requireSession(a.newNoteGetHandler))
	mux.HandleFunc("POST /notes", requireSession(a.createNoteHandler))
	mux.HandleFunc("GET /notes/{short_id}", requireSession(a.viewNoteHandler))
	mux.HandleFunc("GET /notes/{short_id}/edit", requireSession(a.editNoteGetHandler))
	mux.HandleFunc("POST /notes/{short_id}", requireSession(a.updateNoteHandler))
	mux.HandleFunc("POST /notes/{short_id}/delete", requireSession(a.deleteNoteHandler))

	mux.HandleFunc("GET /api/notes", requireAPIKey(a.apiListNotes))
	mux.HandleFunc("POST /api/notes", requireAPIKey(a.apiCreateNote))
	mux.HandleFunc("GET /api/notes/{short_id}", requireAPIKey(a.apiGetNote))
	mux.HandleFunc("PUT /api/notes/{short_id}", requireAPIKey(a.apiUpdateNote))
	mux.HandleFunc("DELETE /api/notes/{short_id}", requireAPIKey(a.apiDeleteNote))

	return mux
}
