package main

import (
	"context"
	"html/template"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/stevedylandev/andromeda/crates-go/auth"
	"github.com/stevedylandev/andromeda/crates-go/config"
	"github.com/stevedylandev/andromeda/crates-go/sqlite"
)

func main() {
	config.LoadDotEnv(".env")
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	dbPath := config.Getenv("BOOKMARKS_DB_PATH", "bookmarks.sqlite")
	db, err := sqlite.Open(dbPath, schema)
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
	app := &App{
		DB:           db,
		Log:          logger,
		Templates:    tmpl,
		Sessions:     sessions,
		Password:     os.Getenv("BOOKMARKS_PASSWORD"),
		APIKey:       os.Getenv("BOOKMARKS_API_KEY"),
		CookieSecure: sessions.CookieSecure,
	}

	go app.faviconBackfill(context.Background())

	addr := config.Getenv("HOST", "127.0.0.1") + ":" + config.Getenv("PORT", "3000")
	logger.Info("bookmarks server running", "addr", addr)
	if err := http.ListenAndServe(addr, app.routes()); err != nil {
		log.Fatal(err)
	}
}

func (a *App) faviconBackfill(ctx context.Context) {
	pending, err := listLinksMissingFavicon(a.DB)
	if err != nil {
		a.Log.Error("favicon backfill query", "err", err)
		return
	}
	if len(pending) == 0 {
		return
	}
	a.Log.Info("favicon backfill", "count", len(pending))
	for _, row := range pending {
		if fav := discoverFavicon(ctx, row.URL); fav != "" {
			if err := updateLinkFavicon(a.DB, row.ID, &fav); err != nil {
				a.Log.Error("favicon backfill update", "id", row.ID, "err", err)
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	a.Log.Info("favicon backfill done")
}
