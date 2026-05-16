package main

import (
	"context"
	"net/http"
	"strings"

	"github.com/stevedylandev/andromeda/crates-go/auth"
	"github.com/stevedylandev/andromeda/crates-go/web"
)

func (a *App) indexHandler(w http.ResponseWriter, r *http.Request) {
	cats, _ := listCategories(a.DB)
	links, _ := listLinks(a.DB)
	groups := make([]categoryGroup, 0, len(cats))
	for _, c := range cats {
		group := categoryGroup{Name: c.Name}
		for _, l := range links {
			if l.CategoryID == c.ID {
				group.Links = append(group.Links, l)
			}
		}
		groups = append(groups, group)
	}
	web.Render(a.Templates, w, "index.html", indexPageData{Groups: groups}, a.Log)
}

func (a *App) loginGetHandler(w http.ResponseWriter, r *http.Request) {
	web.Render(a.Templates, w, "login.html", loginPageData{Error: r.URL.Query().Get("error")}, a.Log)
}

func (a *App) loginPostHandler(w http.ResponseWriter, r *http.Request) {
	if a.Password == "" {
		web.RedirectWithError(w, r, "/login", "No password configured")
		return
	}
	if err := r.ParseForm(); err != nil {
		web.RedirectWithError(w, r, "/login", "Bad request")
		return
	}
	if !auth.VerifyPassword(r.FormValue("password"), a.Password) {
		web.RedirectWithError(w, r, "/login", "Invalid password")
		return
	}
	token, err := a.Sessions.Create()
	if err != nil {
		a.Log.Error("create session failed", "err", err)
		web.RedirectWithError(w, r, "/login", "Session error")
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
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (a *App) adminHandler(w http.ResponseWriter, r *http.Request) {
	cats, _ := listCategories(a.DB)
	raw, _ := listLinks(a.DB)
	catName := map[int64]string{}
	for _, c := range cats {
		catName[c.ID] = c.Name
	}
	rows := make([]adminLinkRow, 0, len(raw))
	for _, l := range raw {
		rows = append(rows, adminLinkRow{ShortID: l.ShortID, Title: l.Title, URL: l.URL, FaviconURL: l.FaviconURL, Category: catName[l.CategoryID]})
	}
	web.Render(a.Templates, w, "admin.html", adminPageData{Success: r.URL.Query().Get("success"), Error: r.URL.Query().Get("error"), Categories: cats, Links: rows}, a.Log)
}

func (a *App) adminAddCategory(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		web.RedirectWithError(w, r, "/admin", "Bad request")
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		web.RedirectWithError(w, r, "/admin", "Name required")
		return
	}
	if _, err := createCategory(a.DB, name); err != nil {
		a.Log.Error("create category", "err", err)
		web.RedirectWithError(w, r, "/admin", "Failed to add category")
		return
	}
	web.RedirectWithSuccess(w, r, "/admin", "Category added")
}

func (a *App) adminDeleteCategory(w http.ResponseWriter, r *http.Request) {
	_, _ = deleteCategoryByShortID(a.DB, r.PathValue("short_id"))
	web.RedirectWithSuccess(w, r, "/admin", "Category removed")
}

func (a *App) adminMoveCategory(w http.ResponseWriter, r *http.Request) {
	dir := 0
	switch r.PathValue("dir") {
	case "up":
		dir = -1
	case "down":
		dir = 1
	default:
		web.RedirectWithError(w, r, "/admin", "Invalid direction")
		return
	}
	if _, err := moveCategory(a.DB, r.PathValue("short_id"), dir); err != nil {
		a.Log.Error("move category", "err", err)
		web.RedirectWithError(w, r, "/admin", "Failed to reorder")
		return
	}
	web.RedirectWithSuccess(w, r, "/admin", "Category reordered")
}

func (a *App) adminAddLink(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		web.RedirectWithError(w, r, "/admin", "Bad request")
		return
	}
	title := strings.TrimSpace(r.FormValue("title"))
	url := strings.TrimSpace(r.FormValue("url"))
	if title == "" || url == "" {
		web.RedirectWithError(w, r, "/admin", "Title and URL required")
		return
	}
	cat, err := getCategoryByName(a.DB, r.FormValue("category"))
	if err != nil || cat == nil {
		web.RedirectWithError(w, r, "/admin", "Unknown category")
		return
	}
	link, err := createLink(a.DB, title, url, nil, cat.ID)
	if err != nil {
		a.Log.Error("create link", "err", err)
		web.RedirectWithError(w, r, "/admin", "Failed to add link")
		return
	}
	go func(id int64, u string) {
		if fav := discoverFavicon(context.Background(), u); fav != "" {
			if err := updateLinkFavicon(a.DB, id, &fav); err != nil {
				a.Log.Error("update favicon", "err", err)
			}
		}
	}(link.ID, url)
	web.RedirectWithSuccess(w, r, "/admin", "Link added")
}

func (a *App) adminDeleteLink(w http.ResponseWriter, r *http.Request) {
	_, _ = deleteLinkByShortID(a.DB, r.PathValue("short_id"))
	web.RedirectWithSuccess(w, r, "/admin", "Link removed")
}
