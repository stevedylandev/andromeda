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

type tagKV struct {
	Key   string
	Value string
}

type linkTag struct {
	Rel   string
	Href  string
	Extra string
}

type resultsData struct {
	URL         string
	Error       string
	OGImage     string
	Favicon     string
	FoundTags   []tagKV
	MissingTags []string
	LinkTags    []linkTag
}
