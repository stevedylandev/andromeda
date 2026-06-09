package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/stevedylandev/andromeda/pkg/config"
)

func main() {
	config.LoadDotEnv(".env")
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	root := config.Getenv("KEPLER_REPO_ROOT", "./repos")
	siteName := config.Getenv("KEPLER_SITE_NAME", "kepler")

	tmpl, err := buildTemplates()
	if err != nil {
		log.Fatal(err)
	}

	app := &App{
		Log:       logger,
		Templates: tmpl,
		RepoRoot:  root,
		SiteName:  siteName,
	}

	addr := config.Getenv("HOST", "127.0.0.1") + ":" + config.Getenv("PORT", "4747")
	logger.Info("kepler server running", "addr", addr, "repo_root", root)
	if err := http.ListenAndServe(addr, app.routes()); err != nil {
		log.Fatal(err)
	}
}
