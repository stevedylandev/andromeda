package main

import (
	"context"
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

	dbPath := getenv("FEEDS_DB_PATH", "feeds.sqlite")
	db, err := openDB(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	defaultPoll := getenvInt("DEFAULT_POLL_MINUTES", 30)
	itemCap := getenvInt("ITEM_CAP_PER_FEED", 200)
	if err := seedSettings(db, defaultPoll); err != nil {
		log.Fatal(err)
	}

	tmpl := template.Must(template.New("").Funcs(template.FuncMap{"safeURL": func(s string) string { return s }}).ParseFS(appFS, "templates/*.html"))
	app := &App{
		DB:                 db,
		Log:                logger,
		Templates:          tmpl,
		AdminPassword:      os.Getenv("ADMIN_PASSWORD"),
		APIKey:             os.Getenv("API_KEY"),
		CookieSecure:       strings.EqualFold(os.Getenv("COOKIE_SECURE"), "true"),
		BaseURL:            getenv("BASE_URL", "http://localhost:3000"),
		DefaultPollMinutes: defaultPoll,
		ItemCap:            itemCap,
	}
	if app.APIKey == "" {
		logger.Warn("API_KEY is not set; API requires session cookie only")
	}
	go app.poller(context.Background())

	addr := getenv("HOST", "0.0.0.0") + ":" + getenv("PORT", "3000")
	logger.Info("feeds-go server running", "addr", addr)
	if err := http.ListenAndServe(addr, app.routes()); err != nil {
		log.Fatal(err)
	}
}
