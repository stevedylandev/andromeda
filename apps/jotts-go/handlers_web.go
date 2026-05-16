package main

import (
	"html/template"
	"net/http"
	"strings"
	"time"
)

func (a *App) loginGetHandler(w http.ResponseWriter, r *http.Request) {
	a.render(w, "login.html", loginPageData{Error: r.URL.Query().Get("error")})
}

func (a *App) loginPostHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/login?error=Invalid+request", http.StatusSeeOther)
		return
	}
	password := r.FormValue("password")
	if !secureEqual(password, a.Password) {
		http.Redirect(w, r, "/login?error=Invalid+password", http.StatusSeeOther)
		return
	}
	token, err := generateSessionToken()
	if err != nil {
		a.Log.Error("session token failed", "err", err)
		http.Redirect(w, r, "/login?error=Server+error", http.StatusSeeOther)
		return
	}
	if err := createSession(a.DB, token, time.Now().UTC().Add(7*24*time.Hour)); err != nil {
		a.Log.Error("create session failed", "err", err)
		http.Redirect(w, r, "/login?error=Server+error", http.StatusSeeOther)
		return
	}
	http.SetCookie(w, a.sessionCookie(token))
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) logoutHandler(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookieName); err == nil && c.Value != "" {
		deleteSession(a.DB, c.Value)
	}
	http.SetCookie(w, a.clearSessionCookie())
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (a *App) indexHandler(w http.ResponseWriter, r *http.Request) {
	notes, err := listNotes(a.DB)
	if err != nil {
		a.Log.Error("list notes failed", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	a.render(w, "index.html", indexPageData{Notes: notes})
}

func (a *App) newNoteGetHandler(w http.ResponseWriter, r *http.Request) {
	a.render(w, "new.html", newPageData{Error: r.URL.Query().Get("error")})
}

func (a *App) createNoteHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		redirectWithError(w, r, "/notes/new", "Invalid request")
		return
	}
	title := strings.TrimSpace(r.FormValue("title"))
	content := r.FormValue("content")
	if title == "" {
		redirectWithError(w, r, "/notes/new", "Title is required")
		return
	}
	note, err := createNote(a.DB, title, content)
	if err != nil {
		a.Log.Error("create note failed", "err", err)
		redirectWithError(w, r, "/notes/new", "Failed to create note")
		return
	}
	http.Redirect(w, r, "/notes/"+note.ShortID, http.StatusSeeOther)
}

func (a *App) viewNoteHandler(w http.ResponseWriter, r *http.Request) {
	shortID := r.PathValue("short_id")
	note, err := getNoteByShortID(a.DB, shortID)
	if err != nil {
		a.Log.Error("get note failed", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if note == nil {
		http.Error(w, "Note not found", http.StatusNotFound)
		return
	}
	rendered, err := renderMarkdown(note.Content)
	if err != nil {
		a.Log.Error("render markdown failed", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	a.render(w, "view.html", viewPageData{Note: *note, Rendered: template.HTML(rendered)})
}

func (a *App) editNoteGetHandler(w http.ResponseWriter, r *http.Request) {
	shortID := r.PathValue("short_id")
	note, err := getNoteByShortID(a.DB, shortID)
	if err != nil {
		a.Log.Error("get note failed", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if note == nil {
		http.Error(w, "Note not found", http.StatusNotFound)
		return
	}
	a.render(w, "edit.html", editPageData{Note: *note, Error: r.URL.Query().Get("error")})
}

func (a *App) updateNoteHandler(w http.ResponseWriter, r *http.Request) {
	shortID := r.PathValue("short_id")
	if err := r.ParseForm(); err != nil {
		redirectWithError(w, r, "/notes/"+shortID+"/edit", "Invalid request")
		return
	}
	title := strings.TrimSpace(r.FormValue("title"))
	content := r.FormValue("content")
	if title == "" {
		redirectWithError(w, r, "/notes/"+shortID+"/edit", "Title is required")
		return
	}
	note, err := updateNoteByShortID(a.DB, shortID, title, content)
	if err != nil {
		a.Log.Error("update note failed", "err", err)
		redirectWithError(w, r, "/notes/"+shortID+"/edit", "Failed to update note")
		return
	}
	if note == nil {
		http.Error(w, "Note not found", http.StatusNotFound)
		return
	}
	http.Redirect(w, r, "/notes/"+shortID, http.StatusSeeOther)
}

func (a *App) deleteNoteHandler(w http.ResponseWriter, r *http.Request) {
	shortID := r.PathValue("short_id")
	if _, err := deleteNoteByShortID(a.DB, shortID); err != nil {
		a.Log.Error("delete note failed", "err", err)
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
