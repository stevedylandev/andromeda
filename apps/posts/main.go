package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"

	poststorage "github.com/stevedylandev/andromeda/apps/posts/storage"
	"github.com/stevedylandev/andromeda/pkg/auth"
	"github.com/stevedylandev/andromeda/pkg/config"
	"github.com/stevedylandev/andromeda/pkg/sqlite"
)

func main() {
	config.LoadDotEnv(".env")
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	dbPath := config.Getenv("POSTS_DB_PATH", "posts.sqlite")
	db, err := sqlite.Open(dbPath, postsSchema)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err := runMigrations(db); err != nil {
		log.Fatalf("migrations: %v", err)
	}
	seedDefaultSettings(db)

	uploadsDir := config.Getenv("UPLOADS_DIR", "uploads")
	if err := ensureDir(uploadsDir); err != nil {
		log.Fatalf("create uploads dir: %v", err)
	}
	storageBackend := poststorage.Backend(poststorage.NewLocal(uploadsDir))
	if os.Getenv("R2_BUCKET") != "" {
		r2, err := poststorage.NewR2(
			os.Getenv("R2_ACCOUNT_ID"),
			os.Getenv("R2_ACCESS_KEY_ID"),
			os.Getenv("R2_SECRET_ACCESS_KEY"),
			os.Getenv("R2_BUCKET"),
			os.Getenv("R2_PUBLIC_URL"),
		)
		if err != nil {
			log.Fatalf("configure r2: %v", err)
		}
		storageBackend = r2
		logger.Info("using R2 storage backend", "bucket", os.Getenv("R2_BUCKET"))
	} else {
		logger.Info("using local storage backend", "dir", uploadsDir)
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

	tmpl, err := buildTemplates()
	if err != nil {
		log.Fatal(err)
	}
	app := &App{
		DB:           db,
		Log:          logger,
		Templates:    tmpl,
		Sessions:     sessions,
		AppPassword:  password,
		CookieSecure: sessions.CookieSecure,
		UploadsDir:   uploadsDir,
		Storage:      storageBackend,
		SiteURL:      strings.TrimRight(config.Getenv("SITE_URL", "http://localhost:3000"), "/"),
	}

	addr := config.Getenv("HOST", "127.0.0.1") + ":" + config.Getenv("PORT", "3000")
	logger.Info("posts server running", "addr", addr)
	if err := http.ListenAndServe(addr, app.routes()); err != nil {
		log.Fatal(err)
	}
}
