package main

import (
	"context"
	"html/template"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/stevedylandev/andromeda/crates-go/auth"
	"github.com/stevedylandev/andromeda/crates-go/config"
	"github.com/stevedylandev/andromeda/crates-go/sqlite"

	_ "golang.org/x/crypto/x509roots/fallback"
)

func main() {
	config.LoadDotEnv(".env")
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	dbPath := config.Getenv("FEEDS_DB_PATH", "feeds.sqlite")
	db, err := sqlite.Open(dbPath, feedsSchema)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	defaultPoll := config.GetenvInt("DEFAULT_POLL_MINUTES", 30)
	itemCap := config.GetenvInt("ITEM_CAP_PER_FEED", 200)
	if err := seedSettings(db, defaultPoll); err != nil {
		log.Fatal(err)
	}

	sessions := &auth.Store{DB: db, CookieName: "feeds_session", CookieSecure: config.GetenvBool("COOKIE_SECURE", false)}
	if err := sessions.EnsureSchema(); err != nil {
		log.Fatal(err)
	}
	sessions.PruneExpired()

	tmpl := template.Must(template.New("").Funcs(template.FuncMap{"safeURL": func(s string) string { return s }}).ParseFS(appFS, "templates/*.html"))
	app := &App{
		DB:                 db,
		Log:                logger,
		Templates:          tmpl,
		Sessions:           sessions,
		AdminPassword:      os.Getenv("ADMIN_PASSWORD"),
		APIKey:             os.Getenv("API_KEY"),
		CookieSecure:       sessions.CookieSecure,
		BaseURL:            config.Getenv("BASE_URL", "http://localhost:3000"),
		DefaultPollMinutes: defaultPoll,
		ItemCap:            itemCap,
	}
	if app.APIKey == "" {
		logger.Warn("API_KEY is not set; API requires session cookie only")
	}
	go app.poller(context.Background())

	addr := config.Getenv("HOST", "0.0.0.0") + ":" + config.Getenv("PORT", "3000")
	logger.Info("feeds server running", "addr", addr)
	if err := http.ListenAndServe(addr, app.routes()); err != nil {
		log.Fatal(err)
	}
}
