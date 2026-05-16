package main

import (
	"html/template"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/stevedylandev/andromeda/crates-go/auth"
	"github.com/stevedylandev/andromeda/crates-go/config"
)

func runServer(args []string) {
	config.LoadDotEnv(".env")
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	dbPath := config.Getenv("JOTTS_DB_PATH", "jotts.sqlite")
	db, err := openDB(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	sessions := &auth.Store{DB: db, CookieName: "session", CookieSecure: config.GetenvBool("COOKIE_SECURE", false)}
	if err := sessions.EnsureSchema(); err != nil {
		log.Fatal(err)
	}
	sessions.PruneExpired()

	tmpl := template.Must(template.ParseFS(appFS, "templates/*.html"))

	password := os.Getenv("JOTTS_PASSWORD")
	if password == "" {
		logger.Warn("JOTTS_PASSWORD not set, using default 'changeme'")
		password = "changeme"
	}
	apiKey := os.Getenv("JOTTS_API_KEY")
	if apiKey == "" {
		logger.Info("JOTTS_API_KEY not set, /api/* will return 403")
	}

	app := &App{
		DB:           db,
		Log:          logger,
		Templates:    tmpl,
		Sessions:     sessions,
		Password:     password,
		APIKey:       apiKey,
		CookieSecure: sessions.CookieSecure,
	}

	addr := config.Getenv("HOST", "127.0.0.1") + ":" + config.Getenv("PORT", "3000")
	logger.Info("jotts-go server running", "addr", addr)
	if err := http.ListenAndServe(addr, app.routes()); err != nil {
		log.Fatal(err)
	}
}
