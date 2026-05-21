package main

import (
	"net/http"
	"strings"

	"github.com/stevedylandev/andromeda/crates-go/web"
)

func (a *App) apiListNotes(w http.ResponseWriter, r *http.Request) {
	notes, err := listNotes(a.DB)
	if err != nil {
		web.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if notes == nil {
		notes = []Note{}
	}
	web.WriteJSON(w, http.StatusOK, notes)
}

func (a *App) apiGetNote(w http.ResponseWriter, r *http.Request) {
	shortID := r.PathValue("short_id")
	note, err := getNoteByShortID(a.DB, shortID)
	if err != nil {
		web.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if note == nil {
		http.NotFound(w, r)
		return
	}
	web.WriteJSON(w, http.StatusOK, note)
}

func (a *App) apiCreateNote(w http.ResponseWriter, r *http.Request) {
	var body NoteInput
	if !web.DecodeJSON(w, r, &body) {
		return
	}
	title := strings.TrimSpace(body.Title)
	if title == "" {
		http.Error(w, "title required", http.StatusBadRequest)
		return
	}
	note, err := createNote(a.DB, title, body.Content)
	if err != nil {
		web.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	web.WriteJSON(w, http.StatusCreated, note)
}

func (a *App) apiUpdateNote(w http.ResponseWriter, r *http.Request) {
	shortID := r.PathValue("short_id")
	var body NoteInput
	if !web.DecodeJSON(w, r, &body) {
		return
	}
	title := strings.TrimSpace(body.Title)
	if title == "" {
		http.Error(w, "title required", http.StatusBadRequest)
		return
	}
	note, err := updateNoteByShortID(a.DB, shortID, title, body.Content)
	if err != nil {
		web.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if note == nil {
		http.NotFound(w, r)
		return
	}
	web.WriteJSON(w, http.StatusOK, note)
}

func (a *App) apiDeleteNote(w http.ResponseWriter, r *http.Request) {
	shortID := r.PathValue("short_id")
	ok, err := deleteNoteByShortID(a.DB, shortID)
	if err != nil {
		web.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
