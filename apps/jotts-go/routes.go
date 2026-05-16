package main

import "net/http"

func (a *App) routes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /static/", a.embeddedHandler("static"))
	mux.HandleFunc("GET /assets/", a.embeddedHandler("assets"))

	mux.HandleFunc("GET /login", a.loginGetHandler)
	mux.HandleFunc("POST /login", a.loginPostHandler)
	mux.HandleFunc("GET /logout", a.logoutHandler)

	mux.HandleFunc("GET /{$}", a.requireSession(a.indexHandler))
	mux.HandleFunc("GET /notes/new", a.requireSession(a.newNoteGetHandler))
	mux.HandleFunc("POST /notes", a.requireSession(a.createNoteHandler))
	mux.HandleFunc("GET /notes/{short_id}", a.requireSession(a.viewNoteHandler))
	mux.HandleFunc("GET /notes/{short_id}/edit", a.requireSession(a.editNoteGetHandler))
	mux.HandleFunc("POST /notes/{short_id}", a.requireSession(a.updateNoteHandler))
	mux.HandleFunc("POST /notes/{short_id}/delete", a.requireSession(a.deleteNoteHandler))

	mux.HandleFunc("GET /api/notes", a.requireAPIKey(a.apiListNotes))
	mux.HandleFunc("POST /api/notes", a.requireAPIKey(a.apiCreateNote))
	mux.HandleFunc("GET /api/notes/{short_id}", a.requireAPIKey(a.apiGetNote))
	mux.HandleFunc("PUT /api/notes/{short_id}", a.requireAPIKey(a.apiUpdateNote))
	mux.HandleFunc("DELETE /api/notes/{short_id}", a.requireAPIKey(a.apiDeleteNote))

	return mux
}
