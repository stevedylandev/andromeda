package main

import (
	"html/template"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/stevedylandev/andromeda/pkg/auth"
	"github.com/stevedylandev/andromeda/pkg/config"
	"github.com/stevedylandev/andromeda/pkg/sqlite"
)

func main() {
	config.LoadDotEnv(".env")
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	dbPath := config.Getenv("HABBITS_DB_PATH", "habbits.sqlite")

	db, err := sqlite.Open(dbPath, habbitsSchema)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	sessions := &auth.Store{DB: db, CookieName: "session", CookieSecure: config.GetenvBool("COOKIE_SECURE", false)}
	if err := sessions.EnsureSchema(); err != nil {
		log.Fatal(err)
	}
	sessions.PruneExpired()

	adminPassword := config.Getenv("HABBITS_PASSWORD", "")
	if adminPassword == "" {
		logger.Warn("HABBITS_PASSWORD not set; admin login is disabled")
	}

	tmpl := template.Must(template.ParseFS(appFS, "templates/*.html"))
	app := &App{
		DB:            db,
		Log:           logger,
		Templates:     tmpl,
		Sessions:      sessions,
		AdminPassword: adminPassword,
		APIKey:        config.Getenv("HABBITS_API_KEY", ""),
		CookieSecure:  sessions.CookieSecure,
		BaseURL:       config.Getenv("BASE_URL", "http://localhost:3000"),
	}

	addr := config.Getenv("HOST", "0.0.0.0") + ":" + config.Getenv("PORT", "3000")
	logger.Info("habbits server running", "addr", addr)
	if err := http.ListenAndServe(addr, app.routes()); err != nil {
		log.Fatal(err)
	}
}
