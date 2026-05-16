package main

import (
	"embed"
	"html/template"
	"log/slog"
)

//go:embed templates/*.html static/*
var appFS embed.FS

type App struct {
	Log       *slog.Logger
	Templates *template.Template
}
