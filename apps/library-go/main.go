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

func main() {
	config.LoadDotEnv(".env")
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	dbPath := config.Getenv("LIBRARY_DB_PATH", "library.sqlite")
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
	app := &App{
		DB:             db,
		Log:            logger,
		Templates:      tmpl,
		Sessions:       sessions,
		AdminPassword:  os.Getenv("ADMIN_PASSWORD"),
		GoogleBooksKey: os.Getenv("GOOGLE_BOOKS_API_KEY"),
		CookieSecure:   sessions.CookieSecure,
		BaseURL:        config.Getenv("BASE_URL", "http://localhost:3000"),
		DisplayMode:    parseDisplayMode(config.Getenv("LIBRARY_DISPLAY_MODE", "inline")),
	}

	addr := config.Getenv("HOST", "0.0.0.0") + ":" + config.Getenv("PORT", "3000")
	logger.Info("library-go server running", "addr", addr)
	if err := http.ListenAndServe(addr, app.routes()); err != nil {
		log.Fatal(err)
	}
}
