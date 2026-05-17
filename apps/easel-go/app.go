package main

import (
	"database/sql"
	"embed"
	"html/template"
	"log/slog"
	"net/http"
	"time"
)

//go:embed templates/*.html static/*
var appFS embed.FS

type App struct {
	DB              *sql.DB
	Log             *slog.Logger
	Templates       map[string]*template.Template
	HTTP            *http.Client
	TZ              *time.Location
	TZName          string
	Classifications []string
	ExcludeTerms    []string
	BackfillDays    int
	MaxDedupRetries int
	BaseURL         string
}

type artworkView struct {
	Date             string
	Title            string
	ArtistDisplay    string
	DateDisplay      string
	MediumDisplay    string
	Dimensions       string
	PlaceOfOrigin    string
	CreditLine       string
	Description      string
	DescriptionHTML  template.HTML
	ShortDescription string
	ImageURL         string
	SourceURL        string
}

type archiveRow struct {
	Date   string
	Title  string
	Artist string
}

type indexPageData struct {
	TodayDate string
	Artwork   *artworkView
}

type dayPageData struct {
	Date    string
	Artwork artworkView
}

type archivePageData struct {
	Archive []archiveRow
}

type errorPageData struct {
	Title   string
	Message string
}
