package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/stevedylandev/andromeda/crates-go/config"
)

func main() {
	config.LoadDotEnv(".env")
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	tmpl, err := buildTemplates()
	if err != nil {
		log.Fatal(err)
	}
	app := &App{Log: logger, Templates: tmpl}

	addr := config.Getenv("HOST", "0.0.0.0") + ":" + config.Getenv("PORT", "3000")
	logger.Info("og server running", "addr", addr)
	if err := http.ListenAndServe(addr, app.routes()); err != nil {
		log.Fatal(err)
	}
}
