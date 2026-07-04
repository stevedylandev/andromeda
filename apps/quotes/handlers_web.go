package main

import (
	"net/http"
	"strings"

	"github.com/stevedylandev/andromeda/pkg/auth"
	"github.com/stevedylandev/andromeda/pkg/web"
)

func quoteToView(q Quote) quoteView {
	v := quoteView{Text: q.Text, Author: q.Author}
	if q.Source != nil {
		v.Source = *q.Source
	}
	return v
}

func quoteToRow(q Quote) adminQuoteRow {
	r := adminQuoteRow{ShortID: q.ShortID, Text: q.Text, Author: q.Author}
	if q.Source != nil {
		r.Source = *q.Source
	}
	return r
}

func (a *App) indexHandler(w http.ResponseWriter, r *http.Request) {
	data := indexPageData{BaseURL: a.BaseURL}
	if q, err := quoteOfTheDay(a.DB); err != nil {
		a.Log.Error("quote of the day", "err", err)
	} else if q != nil {
		v := quoteToView(*q)
		data.Quote = &v
	}
	web.Render(a.Templates, w, "index.html", data, a.Log)
}

func (a *App) randomHandler(w http.ResponseWriter, r *http.Request) {
	data := indexPageData{BaseURL: a.BaseURL}
	if q, err := randomQuote(a.DB); err != nil {
		a.Log.Error("random quote", "err", err)
	} else if q != nil {
		v := quoteToView(*q)
		data.Quote = &v
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
	total, _ := countQuotes(a.DB)

	query := r.URL.Query().Get("q")
	searched := strings.TrimSpace(query) != ""

	var found []Quote
	if searched {
		found, _ = searchQuotes(a.DB, query)
	} else {
		found, _ = listQuotes(a.DB, 50)
	}
	rows := make([]adminQuoteRow, 0, len(found))
	for _, q := range found {
		rows = append(rows, quoteToRow(q))
	}

	web.Render(a.Templates, w, "admin.html", adminPageData{
		Success:  r.URL.Query().Get("success"),
		Error:    r.URL.Query().Get("error"),
		Total:    total,
		Quotes:   rows,
		Query:    query,
		Searched: searched,
	}, a.Log)
}

func (a *App) adminAddQuote(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		web.RedirectWithError(w, r, "/admin", "Bad request")
		return
	}
	text := strings.TrimSpace(r.FormValue("text"))
	author := strings.TrimSpace(r.FormValue("author"))
	source := strings.TrimSpace(r.FormValue("source"))
	if text == "" || author == "" {
		web.RedirectWithError(w, r, "/admin", "Quote and author are required")
		return
	}
	if _, err := insertQuote(a.DB, text, author, source); err != nil {
		a.Log.Error("insert quote", "err", err)
		web.RedirectWithError(w, r, "/admin", "Failed to add quote")
		return
	}
	web.RedirectWithSuccess(w, r, "/admin", "Quote added")
}

func (a *App) adminDeleteQuote(w http.ResponseWriter, r *http.Request) {
	if err := deleteQuoteByShortID(a.DB, r.PathValue("short_id")); err != nil {
		a.Log.Error("delete quote", "err", err)
		web.RedirectWithError(w, r, "/admin", "Failed to remove quote")
		return
	}
	web.RedirectWithSuccess(w, r, "/admin", "Quote removed")
}
