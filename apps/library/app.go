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

type DisplayMode int

const (
	DisplayModeInline DisplayMode = iota
	DisplayModeNav
)

func parseDisplayMode(s string) DisplayMode {
	if s == "nav" {
		return DisplayModeNav
	}
	return DisplayModeInline
}

type App struct {
	DB            *sql.DB
	Log           *slog.Logger
	Templates     *template.Template
	Sessions      *auth.Store
	AdminPassword string
	CookieSecure  bool
	BaseURL       string
	DisplayMode   DisplayMode
}

type bookView struct {
	ID       int64
	Title    string
	Authors  string
	ISBN     string
	CoverURL string
	Notes    string
}

type quoteView struct {
	ID     int64
	BookID int64
	Page   string
	Quote  string
}

type quoteGroup struct {
	BookID      int64
	BookTitle   string
	BookAuthors string
	Quotes      []quoteView
}

type sectionView struct {
	Label string
	Books []bookView
}

type navCategory struct {
	Slug   string
	Label  string
	Active bool
}

type indexPageData struct {
	BaseURL       string
	Sections      []sectionView
	NavMode       bool
	NavCategories []navCategory
}

type loginPageData struct {
	Error string
}

type bookPageData struct {
	BaseURL string
	Book    bookView
	Quotes  []quoteView
}

type quotesPageData struct {
	BaseURL string
	Groups  []quoteGroup
}

type quotePageData struct {
	BaseURL string
	Quote   quoteView
	Book    bookView
}

type adminBookRow struct {
	ID       int64
	Title    string
	Authors  string
	ISBN     string
	CoverURL string
	Notes    string
	Status   string
}

type adminQuoteRow struct {
	ID          int64
	BookID      int64
	Page        string
	Quote       string
	BookTitle   string
	BookAuthors string
}

type bookWithQuotesPageData struct {
	BaseURL string
	Success string
	Error   string
	Book    adminBookRow
	Quotes  []adminQuoteRow
}

type adminQuotesPageData struct {
	BaseURL string
	Success string
	Error   string
	Quotes  []adminQuoteRow
	Books   []adminBookRow
}

type adminPageData struct {
	Success         string
	Error           string
	Books           []adminBookRow
	Labels          CategoryLabels
	LibraryQuery    string
	LibraryResults  []adminBookRow
	LibrarySearched bool
}
