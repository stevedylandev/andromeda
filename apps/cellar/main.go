package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/stevedylandev/andromeda/crates-go/auth"
	"github.com/stevedylandev/andromeda/crates-go/config"
	"github.com/stevedylandev/andromeda/crates-go/sqlite"
)

func main() {
	config.LoadDotEnv(".env")
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	dbPath := config.Getenv("CELLAR_DB_PATH", "cellar.sqlite")
	db, err := sqlite.Open(dbPath, cellarSchema)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	password := os.Getenv("CELLAR_PASSWORD")
	if password == "" {
		logger.Warn("CELLAR_PASSWORD not set, using default 'changeme'")
		password = "changeme"
	}

	sessions := &auth.Store{DB: db, CookieName: "session", CookieSecure: config.GetenvBool("COOKIE_SECURE", false)}
	if err := sessions.EnsureSchema(); err != nil {
		log.Fatal(err)
	}
	sessions.PruneExpired()

	tmpl, err := buildTemplates()
	if err != nil {
		log.Fatal(err)
	}
	app := &App{
		DB:              db,
		Log:             logger,
		Templates:       tmpl,
		Sessions:        sessions,
		AppPassword:     password,
		CookieSecure:    sessions.CookieSecure,
		AnthropicAPIKey: os.Getenv("ANTHROPIC_API_KEY"),
		SiteURL:         strings.TrimRight(config.Getenv("SITE_URL", "http://localhost:3000"), "/"),
		SiteTitle:       config.Getenv("SITE_TITLE", "Cellar"),
		SiteDescription: config.Getenv("SITE_DESCRIPTION", "Personal wine tasting log"),
	}

	addr := config.Getenv("HOST", "127.0.0.1") + ":" + config.Getenv("PORT", "3000")
	logger.Info("cellar server running", "addr", addr)
	if err := http.ListenAndServe(addr, app.routes()); err != nil {
		log.Fatal(err)
	}
}
