package main

import (
	"html/template"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/stevedylandev/andromeda/crates-go/auth"
	"github.com/stevedylandev/andromeda/crates-go/config"
)

func main() {
	config.LoadDotEnv(".env")
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	dbPath := config.Getenv("POSTS_DB_PATH", "posts.sqlite")
	db, err := openDB(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	uploadsDir := config.Getenv("UPLOADS_DIR", "uploads")
	if err := ensureDir(uploadsDir); err != nil {
		log.Fatalf("create uploads dir: %v", err)
	}

	password := os.Getenv("POSTS_PASSWORD")
	if password == "" {
		logger.Warn("POSTS_PASSWORD not set, using default 'changeme'")
		password = "changeme"
	}

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
		AppPassword:  password,
		CookieSecure: sessions.CookieSecure,
		UploadsDir:   uploadsDir,
		SiteURL:      strings.TrimRight(config.Getenv("SITE_URL", "http://localhost:3000"), "/"),
	}

	addr := config.Getenv("HOST", "127.0.0.1") + ":" + config.Getenv("PORT", "3000")
	logger.Info("posts-go server running", "addr", addr)
	if err := http.ListenAndServe(addr, app.routes()); err != nil {
		log.Fatal(err)
	}
}
