package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/stevedylandev/andromeda/pkg/auth"
	"github.com/stevedylandev/andromeda/pkg/config"
)

func main() {
	config.LoadDotEnv(".env")
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	dbPath := config.Getenv("BLOBS_DB_PATH", "blobs.sqlite")
	db, err := openDB(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	sessions := &auth.Store{
		DB:           db,
		CookieName:   "blobs_session",
		CookieSecure: config.GetenvBool("BLOBS_COOKIE_SECURE", false),
	}
	if err := sessions.EnsureSchema(); err != nil {
		log.Fatal(err)
	}
	sessions.PruneExpired()
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			sessions.PruneExpired()
		}
	}()

	password := os.Getenv("BLOBS_PASSWORD")
	if password == "" {
		logger.Warn("BLOBS_PASSWORD not set, using default 'changeme'")
		password = "changeme"
	}

	endpoint := config.Getenv("S3_ENDPOINT", "")
	if endpoint == "" {
		if accountID := config.Getenv("R2_ACCOUNT_ID", ""); accountID != "" {
			endpoint = "https://" + accountID + ".r2.cloudflarestorage.com"
		}
	}
	accessKey := config.Getenv("S3_ACCESS_KEY_ID", config.Getenv("R2_ACCESS_KEY_ID", ""))
	secretKey := config.Getenv("S3_SECRET_ACCESS_KEY", config.Getenv("R2_SECRET_ACCESS_KEY", ""))
	region := config.Getenv("S3_REGION", "auto")
	publicURLs := parsePublicURLs(os.Getenv("BLOBS_PUBLIC_URLS"))
	presignTTL := time.Duration(config.GetenvInt("BLOBS_PRESIGN_TTL_SECONDS", 3600)) * time.Second

	client, err := NewS3Client(endpoint, region, accessKey, secretKey, publicURLs, presignTTL)
	if err != nil {
		log.Fatalf("configure S3: %v", err)
	}
	logger.Info("S3 client ready", "endpoint", endpoint, "region", region, "public_buckets", len(publicURLs))

	tmpl, err := buildTemplates()
	if err != nil {
		log.Fatal(err)
	}

	maxUploadMB := int64(config.GetenvInt("BLOBS_MAX_UPLOAD_MB", 100))
	app := &App{
		DB:             db,
		Log:            logger,
		Templates:      tmpl,
		Sessions:       sessions,
		S3:             client,
		Password:       password,
		CookieSecure:   sessions.CookieSecure,
		MaxUploadBytes: maxUploadMB << 20,
	}

	addr := config.Getenv("BLOBS_HOST", "0.0.0.0") + ":" + config.Getenv("BLOBS_PORT", "3000")
	logger.Info("blobs server running", "addr", addr)
	if err := http.ListenAndServe(addr, app.routes()); err != nil {
		log.Fatal(err)
	}
}
