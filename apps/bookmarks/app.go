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
	DB           *sql.DB
	Log          *slog.Logger
	Templates    *template.Template
	Sessions     *auth.Store
	Password     string
	APIKey       string
	CookieSecure bool
}

type Category struct {
	ID       int64  `json:"id"`
	ShortID  string `json:"short_id"`
	Name     string `json:"name"`
	Position int64  `json:"position"`
}

type Link struct {
	ID         int64   `json:"id"`
	ShortID    string  `json:"short_id"`
	Title      string  `json:"title"`
	URL        string  `json:"url"`
	FaviconURL *string `json:"favicon_url,omitempty"`
	CategoryID int64   `json:"category_id"`
	CreatedAt  int64   `json:"created_at"`
}

type adminLinkRow struct {
	ShortID    string
	Title      string
	URL        string
	FaviconURL *string
	Category   string
}

type categoryGroup struct {
	Name  string
	Links []Link
}

type indexPageData struct {
	Groups []categoryGroup
}

type loginPageData struct {
	Error string
}

type adminPageData struct {
	Success    string
	Error      string
	Categories []Category
	Links      []adminLinkRow
}

type apiCreateLinkBody struct {
	Category string `json:"category"`
	Title    string `json:"title"`
	URL      string `json:"url"`
}
