package main

import (
	"net/http"

	"github.com/stevedylandev/andromeda/pkg/auth"
)

func (a *App) rootRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/buckets", http.StatusSeeOther)
}

func (a *App) loginGet(w http.ResponseWriter, r *http.Request) {
	if a.Sessions.HasValid(r) {
		http.Redirect(w, r, "/buckets", http.StatusSeeOther)
		return
	}
	a.renderPage(w, "login.html", loginPageData{Error: r.URL.Query().Get("error")})
}

func (a *App) loginPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/login?error=Bad+request", http.StatusSeeOther)
		return
	}
	if !auth.VerifyPassword(r.FormValue("password"), a.Password) {
		http.Redirect(w, r, "/login?error=Invalid+password", http.StatusSeeOther)
		return
	}
	token, err := a.Sessions.Create()
	if err != nil {
		a.Log.Error("create session", "err", err)
		http.Redirect(w, r, "/login?error=Server+error", http.StatusSeeOther)
		return
	}
	a.Sessions.PruneExpired()
	http.SetCookie(w, a.Sessions.SessionCookie(token))
	http.Redirect(w, r, "/buckets", http.StatusSeeOther)
}

func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(a.Sessions.CookieName); err == nil && c.Value != "" {
		a.Sessions.Delete(c.Value)
	}
	http.SetCookie(w, a.Sessions.ClearCookie())
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
