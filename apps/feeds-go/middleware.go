package main

import (
	"net/http"
	"strings"
)

func (a *App) requireSession(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !a.hasValidSession(r) {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

func (a *App) requireAPIAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if a.APIKey != "" {
			authz := r.Header.Get("Authorization")
			if strings.HasPrefix(strings.ToLower(authz), "bearer ") && verifyAPIKey(strings.TrimSpace(authz[7:]), a.APIKey) {
				next(w, r)
				return
			}
		}
		if a.hasValidSession(r) {
			next(w, r)
			return
		}
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
	}
}

func (a *App) hasValidSession(r *http.Request) bool {
	cookie, err := r.Cookie("feeds_session")
	if err != nil || cookie.Value == "" {
		return false
	}
	return isValidSession(a.DB, cookie.Value)
}

func (a *App) sessionCookie(token string) *http.Cookie {
	return &http.Cookie{Name: "feeds_session", Value: token, Path: "/", HttpOnly: true, Secure: a.CookieSecure, SameSite: http.SameSiteLaxMode, MaxAge: 7 * 24 * 60 * 60}
}

func (a *App) withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}
