package main

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/stevedylandev/andromeda/crates-go/auth"
	"github.com/stevedylandev/andromeda/crates-go/web"
)

var sectionDefs = []struct {
	Slug   string
	Status string
}{
	{"reading", "reading"},
	{"read", "read"},
	{"want", "want"},
}

func bookToView(b Book) bookView {
	v := bookView{Title: b.Title, Authors: b.Authors}
	if b.CoverURL != nil {
		v.CoverURL = *b.CoverURL
	}
	if b.Notes != nil {
		v.Notes = *b.Notes
	}
	return v
}

func bookToRow(b Book) adminBookRow {
	r := adminBookRow{ID: b.ID, Title: b.Title, Authors: b.Authors, Status: b.Status}
	if b.ISBN != nil {
		r.ISBN = *b.ISBN
	}
	if b.CoverURL != nil {
		r.CoverURL = *b.CoverURL
	}
	if b.Notes != nil {
		r.Notes = *b.Notes
	}
	return r
}

func (a *App) indexHandler(w http.ResponseWriter, r *http.Request) {
	all, _ := listBooks(a.DB, "")
	labels, _ := getCategoryLabels(a.DB)

	makeSection := func(status string) sectionView {
		sec := sectionView{Label: labelFor(labels, status)}
		for _, b := range all {
			if b.Status == status {
				sec.Books = append(sec.Books, bookToView(b))
			}
		}
		return sec
	}

	data := indexPageData{BaseURL: a.BaseURL, NavMode: a.DisplayMode == DisplayModeNav}
	if data.NavMode {
		selected := sectionDefs[0].Slug
		if q := r.URL.Query().Get("category"); q != "" {
			for _, s := range sectionDefs {
				if s.Slug == q {
					selected = q
					break
				}
			}
		}
		for _, s := range sectionDefs {
			data.NavCategories = append(data.NavCategories, navCategory{Slug: s.Slug, Label: labelFor(labels, s.Status), Active: s.Slug == selected})
		}
		for _, s := range sectionDefs {
			if s.Slug == selected {
				data.Sections = []sectionView{makeSection(s.Status)}
				break
			}
		}
	} else {
		for _, s := range sectionDefs {
			sec := makeSection(s.Status)
			if len(sec.Books) > 0 {
				data.Sections = append(data.Sections, sec)
			}
		}
	}
	web.Render(a.Templates, w, "index.html", data, a.Log)
}

func (a *App) loginGetHandler(w http.ResponseWriter, r *http.Request) {
	web.Render(a.Templates, w, "login.html", loginPageData{Error: r.URL.Query().Get("error")}, a.Log)
}

func (a *App) loginPostHandler(w http.ResponseWriter, r *http.Request) {
	if a.AdminPassword == "" {
		web.RedirectWithError(w, r, "/admin/login", "No admin password configured")
		return
	}
	if err := r.ParseForm(); err != nil {
		web.RedirectWithError(w, r, "/admin/login", "Bad request")
		return
	}
	if !auth.VerifyPassword(r.FormValue("password"), a.AdminPassword) {
		web.RedirectWithError(w, r, "/admin/login", "Invalid password")
		return
	}
	token, err := a.Sessions.Create()
	if err != nil {
		a.Log.Error("create session failed", "err", err)
		web.RedirectWithError(w, r, "/admin/login", "Session error")
		return
	}
	a.Sessions.PruneExpired()
	http.SetCookie(w, a.Sessions.SessionCookie(token))
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (a *App) logoutHandler(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(a.Sessions.CookieName); err == nil && c.Value != "" {
		a.Sessions.Delete(c.Value)
	}
	http.SetCookie(w, a.Sessions.ClearCookie())
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

func (a *App) adminHandler(w http.ResponseWriter, r *http.Request) {
	all, _ := listBooks(a.DB, "")
	labels, _ := getCategoryLabels(a.DB)
	rows := make([]adminBookRow, 0, len(all))
	for _, b := range all {
		rows = append(rows, bookToRow(b))
	}

	libraryQuery := r.URL.Query().Get("q")
	searched := strings.TrimSpace(libraryQuery) != ""
	var results []adminBookRow
	if searched {
		found, _ := searchBooks(a.DB, libraryQuery)
		results = make([]adminBookRow, 0, len(found))
		for _, b := range found {
			results = append(results, bookToRow(b))
		}
	}

	web.Render(a.Templates, w, "admin.html", adminPageData{
		Success:         r.URL.Query().Get("success"),
		Error:           r.URL.Query().Get("error"),
		Books:           rows,
		Labels:          labels,
		LibraryQuery:    libraryQuery,
		LibraryResults:  results,
		LibrarySearched: searched,
	}, a.Log)
}

func (a *App) adminUpdateLabels(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		web.RedirectWithError(w, r, "/admin", "Bad request")
		return
	}
	reading := strings.TrimSpace(r.FormValue("reading"))
	read := strings.TrimSpace(r.FormValue("read"))
	want := strings.TrimSpace(r.FormValue("want"))
	if reading == "" || read == "" || want == "" {
		web.RedirectWithError(w, r, "/admin", "Labels cannot be empty")
		return
	}
	if err := setSetting(a.DB, "category_label.reading", reading); err != nil {
		web.RedirectWithError(w, r, "/admin", "Failed to save labels")
		return
	}
	_ = setSetting(a.DB, "category_label.read", read)
	_ = setSetting(a.DB, "category_label.want", want)
	web.RedirectWithSuccess(w, r, "/admin", "Labels updated")
}

func (a *App) adminSearch(w http.ResponseWriter, r *http.Request) {
	hits, err := googleBooksSearch(r.Context(), r.URL.Query().Get("q"), a.GoogleBooksKey)
	if err != nil {
		a.Log.Warn("google books search failed", "err", err)
		web.WriteError(w, http.StatusBadGateway, err.Error())
		return
	}
	if hits == nil {
		hits = []SearchHit{}
	}
	web.WriteJSON(w, http.StatusOK, hits)
}

func (a *App) adminAddBook(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		web.RedirectWithError(w, r, "/admin", "Bad request")
		return
	}
	status := r.FormValue("status")
	if !validStatus(status) {
		web.RedirectWithError(w, r, "/admin", "Invalid status")
		return
	}
	b := NewBook{Title: r.FormValue("title"), Authors: r.FormValue("authors"), Status: status}
	if v := strings.TrimSpace(r.FormValue("google_id")); v != "" {
		b.GoogleID = &v
	}
	if v := strings.TrimSpace(r.FormValue("isbn")); v != "" {
		b.ISBN = &v
	}
	if v := strings.TrimSpace(r.FormValue("cover_url")); v != "" {
		b.CoverURL = &v
	}
	if _, err := insertBook(a.DB, b); err != nil {
		a.Log.Error("insert book", "err", err)
		web.RedirectWithError(w, r, "/admin", "Failed to add book")
		return
	}
	web.RedirectWithSuccess(w, r, "/admin", "Book added")
}

func (a *App) adminUpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		web.RedirectWithError(w, r, "/admin", "Bad id")
		return
	}
	if err := r.ParseForm(); err != nil {
		web.RedirectWithError(w, r, "/admin", "Bad request")
		return
	}
	status := r.FormValue("status")
	if !validStatus(status) {
		web.RedirectWithError(w, r, "/admin", "Invalid status")
		return
	}
	_ = updateBookStatus(a.DB, id, status)
	web.RedirectWithSuccess(w, r, "/admin", "Status updated")
}

func (a *App) adminUpdateNotes(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		web.RedirectWithError(w, r, "/admin", "Bad id")
		return
	}
	if err := r.ParseForm(); err != nil {
		web.RedirectWithError(w, r, "/admin", "Bad request")
		return
	}
	notes := strings.TrimSpace(r.FormValue("notes"))
	var n *string
	if notes != "" {
		n = &notes
	}
	_ = updateBookNotes(a.DB, id, n)
	web.RedirectWithSuccess(w, r, "/admin", "Notes saved")
}

func (a *App) adminDeleteBook(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err == nil {
		_ = deleteBook(a.DB, id)
	}
	web.RedirectWithSuccess(w, r, "/admin", "Book removed")
}
