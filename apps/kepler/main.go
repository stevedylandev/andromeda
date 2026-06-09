package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/stevedylandev/andromeda/pkg/config"
)

func main() {
	config.LoadDotEnv(".env")
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	root := config.Getenv("KEPLER_REPO_ROOT", "./repos")
	siteName := config.Getenv("KEPLER_SITE_NAME", "kepler")
	baseURL := config.Getenv("KEPLER_BASE_URL", "http://localhost:4747")
	cloneBaseURL := config.Getenv("KEPLER_CLONE_BASE_URL", "")
	cloneSSHHost := config.Getenv("KEPLER_CLONE_SSH_HOST", "")

	tmpl, err := buildTemplates()
	if err != nil {
		log.Fatal(err)
	}

	app := &App{
		Log:          logger,
		Templates:    tmpl,
		RepoRoot:     root,
		SiteName:     siteName,
		BaseURL:      strings.TrimRight(baseURL, "/"),
		CloneBaseURL: strings.TrimRight(cloneBaseURL, "/"),
		CloneSSHHost: strings.TrimSpace(cloneSSHHost),
	}

	addr := config.Getenv("HOST", "127.0.0.1") + ":" + config.Getenv("PORT", "4747")
	logger.Info("kepler server running", "addr", addr, "repo_root", root)
	if err := http.ListenAndServe(addr, app.routes()); err != nil {
		log.Fatal(err)
	}
}
