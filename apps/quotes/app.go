package main

import (
	"database/sql"
	"embed"
	"html/template"
	"log/slog"

	"github.com/stevedylandev/andromeda/pkg/auth"
)

//go:embed templates/*.html static/*
var appFS embed.FS

type App struct {
	DB            *sql.DB
	Log           *slog.Logger
	Templates     *template.Template
	Sessions      *auth.Store
	AdminPassword string
	APIKey        string
	CookieSecure  bool
	BaseURL       string
}

type quoteView struct {
	Text   string
	Author string
	Source string
}

type indexPageData struct {
	BaseURL string
	Quote   *quoteView
}

type loginPageData struct {
	Error string
}

type adminQuoteRow struct {
	ShortID string
	Text    string
	Author  string
	Source  string
}

type adminPageData struct {
	Success  string
	Error    string
	Total    int
	Quotes   []adminQuoteRow
	Query    string
	Searched bool
}
