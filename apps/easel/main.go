package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/stevedylandev/andromeda/pkg/config"
	"github.com/stevedylandev/andromeda/pkg/sqlite"
)

func splitCommaTrim(s string) []string {
	out := []string{}
	for _, part := range strings.Split(s, ",") {
		if v := strings.TrimSpace(part); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func main() {
	config.LoadDotEnv(".env")
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	dbPath := config.Getenv("EASEL_DB_PATH", "easel.sqlite")
	db, err := sqlite.Open(dbPath, easelSchema)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	tzName := config.Getenv("EASEL_TIMEZONE", "UTC")
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		logger.Warn("invalid EASEL_TIMEZONE, falling back to UTC", "value", tzName)
		loc = time.UTC
		tzName = "UTC"
	}

	classifications := splitCommaTrim(config.Getenv("EASEL_CLASSIFICATIONS", "painting"))
	if len(classifications) == 0 {
		log.Fatal("EASEL_CLASSIFICATIONS resolved to empty list")
	}
	excludeTerms := splitCommaTrim(config.Getenv("EASEL_EXCLUDE_TERMS", "erotic,erotica,shunga"))

	tmpl, err := buildTemplates()
	if err != nil {
		log.Fatal(err)
	}
	app := &App{
		DB:              db,
		Log:             logger,
		Templates:       tmpl,
		HTTP:            buildHTTPClient(),
		TZ:              loc,
		TZName:          tzName,
		Classifications: classifications,
		ExcludeTerms:    excludeTerms,
		BackfillDays:    config.GetenvInt("EASEL_BACKFILL_DAYS", 0),
		MaxDedupRetries: config.GetenvInt("EASEL_MAX_DEDUP_RETRIES", 10),
		BaseURL:         strings.TrimRight(config.Getenv("EASEL_BASE_URL", "http://localhost:4242"), "/"),
	}
	logger.Info("easel starting", "tz", tzName, "classifications", classifications, "exclude_terms", excludeTerms, "backfill_days", app.BackfillDays, "retries", app.MaxDedupRetries)

	go app.runScheduler(context.Background())

	addr := config.Getenv("HOST", "127.0.0.1") + ":" + config.Getenv("PORT", "4242")
	logger.Info("easel server running", "addr", addr)
	if err := http.ListenAndServe(addr, app.routes()); err != nil {
		log.Fatal(err)
	}
}
