package main

import (
	"html/template"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
)

func main() {
	loadDotEnv(".env")
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	dbPath := getenv("JOTTS_DB_PATH", "jotts.sqlite")
	db, err := openDB(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	pruneExpiredSessions(db)

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
		Password:     password,
		APIKey:       apiKey,
		CookieSecure: strings.EqualFold(os.Getenv("COOKIE_SECURE"), "true"),
	}

	addr := getenv("HOST", "127.0.0.1") + ":" + getenv("PORT", "3000")
	logger.Info("jotts-go server running", "addr", addr)
	if err := http.ListenAndServe(addr, app.routes()); err != nil {
		log.Fatal(err)
	}
}
