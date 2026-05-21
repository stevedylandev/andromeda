package main

import (
	"html/template"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/stevedylandev/andromeda/pkg/config"
)

func main() {
	config.LoadDotEnv(".env")
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	tmpl := template.Must(template.ParseFS(appFS, "templates/*.html"))
	app := &App{Log: logger, Templates: tmpl}

	addr := config.Getenv("HOST", "127.0.0.1") + ":" + config.Getenv("PORT", "3000")
	logger.Info("shrink server running", "addr", addr)
	if err := http.ListenAndServe(addr, app.routes()); err != nil {
		log.Fatal(err)
	}
}
