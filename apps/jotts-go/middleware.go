package main

import (
	"crypto/subtle"
	"net/http"
)

const sessionCookieName = "session"

func (a *App) requireSession(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !a.hasValidSession(r) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

func (a *App) requireAPIKey(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if a.APIKey == "" {
			http.Error(w, "API key not configured on server", http.StatusForbidden)
			return
		}
		provided := r.Header.Get("x-api-key")
		if !secureEqual(provided, a.APIKey) {
			http.Error(w, "Invalid API key", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func (a *App) hasValidSession(r *http.Request) bool {
	c, err := r.Cookie(sessionCookieName)
	if err != nil || c.Value == "" {
		return false
	}
	return isValidSession(a.DB, c.Value)
}

func (a *App) sessionCookie(token string) *http.Cookie {
	return &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   a.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   7 * 24 * 60 * 60,
	}
}

func (a *App) clearSessionCookie() *http.Cookie {
	return &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   a.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	}
}

func secureEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
