package main

import (
	"database/sql"
	"embed"
	"html/template"
	"log/slog"

	"github.com/stevedylandev/andromeda/apps/jotts-go/internal/store"
	"github.com/stevedylandev/andromeda/crates-go/auth"
)

//go:embed templates/*.html static/*
var appFS embed.FS

type App struct {
	DB           *sql.DB
	Log          *slog.Logger
	Templates    *template.Template
	Sessions     *auth.Store
	Password     string
	APIKey       string
	CookieSecure bool
}

type Note = store.Note
type NoteInput = store.NoteInput

type indexPageData struct {
	Notes []Note
}

type loginPageData struct {
	Error string
}

type newPageData struct {
	Error string
}

type editPageData struct {
	Note  Note
	Error string
}

type viewPageData struct {
	Note     Note
	Rendered template.HTML
}
