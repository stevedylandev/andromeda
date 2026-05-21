package main

import (
	"database/sql"
	"embed"
	"html/template"
	"log/slog"

	"github.com/stevedylandev/andromeda/crates-go/auth"
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
	DB             *sql.DB
	Log            *slog.Logger
	Templates      *template.Template
	Sessions       *auth.Store
	AdminPassword  string
	GoogleBooksKey string
	CookieSecure   bool
	BaseURL        string
	DisplayMode    DisplayMode
}

type bookView struct {
	Title    string
	Authors  string
	CoverURL string
	Notes    string
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

type adminBookRow struct {
	ID       int64
	Title    string
	Authors  string
	ISBN     string
	CoverURL string
	Notes    string
	Status   string
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
