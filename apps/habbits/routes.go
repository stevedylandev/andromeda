package main

import (
	"net/http"

	"github.com/stevedylandev/andromeda/pkg/auth"
	"github.com/stevedylandev/andromeda/pkg/darkmatter"
	"github.com/stevedylandev/andromeda/pkg/web"
)

func (a *App) routes() *http.ServeMux {
	mux := http.NewServeMux()

	requireSession := func(next http.HandlerFunc) http.HandlerFunc {
		return a.Sessions.RequireSession("/login", next)
	}
	requireAPIKey := func(next http.HandlerFunc) http.HandlerFunc {
		return auth.RequireAPIKey(a.APIKey, next)
	}

	// Public: only the login flow and static assets.
	mux.HandleFunc("GET /login", a.loginGetHandler)
	mux.HandleFunc("POST /login", a.loginPostHandler)
	mux.HandleFunc("GET /logout", a.logoutHandler)
	mux.HandleFunc("GET /static/", web.EmbeddedHandler(appFS, "static"))
	darkmatter.Mount(mux, "/assets")

	// Everything else requires a session (fully gated admin app).
	mux.HandleFunc("GET /", requireSession(a.dashboardHandler))
	mux.HandleFunc("GET /new", requireSession(a.newHabitHandler))
	mux.HandleFunc("GET /settings", requireSession(a.settingsHandler))
	mux.HandleFunc("GET /export.csv", requireSession(a.exportHandler))

	mux.HandleFunc("POST /habits", requireSession(a.createHabitHandler))
	mux.HandleFunc("GET /habits/{short_id}", requireSession(a.habitDetailHandler))
	mux.HandleFunc("POST /habits/{short_id}", requireSession(a.updateHabitHandler))
	mux.HandleFunc("POST /habits/{short_id}/delete", requireSession(a.deleteHabitHandler))

	mux.HandleFunc("POST /records", requireSession(a.createRecordHandler))
	mux.HandleFunc("POST /records/{short_id}", requireSession(a.updateRecordHandler))
	mux.HandleFunc("POST /records/{short_id}/delete", requireSession(a.deleteRecordHandler))

	// Read-only JSON API, gated by X-API-Key (disabled when key is empty).
	mux.HandleFunc("GET /api/habits", requireAPIKey(a.apiListHabits))
	mux.HandleFunc("GET /api/records", requireAPIKey(a.apiListRecords))

	return mux
}
