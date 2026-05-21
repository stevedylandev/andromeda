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
	DB                 *sql.DB
	Log                *slog.Logger
	Templates          *template.Template
	Sessions           *auth.Store
	AdminPassword      string
	APIKey             string
	CookieSecure       bool
	BaseURL            string
	DefaultPollMinutes int
	ItemCap            int
}

type templateItem struct {
	Title         string
	Link          string
	Author        string
	FormattedDate string
}

type adminSubRow struct {
	ID            int64
	Title         string
	FeedURL       string
	SiteURL       string
	CategoryName  string
	LastFetchedAt string
	LastError     string
}

type indexPageData struct {
	BaseURL  string
	Items    []templateItem
	FeedURLs []string
	Error    string
}

type loginPageData struct {
	Error string
}

type adminPageData struct {
	Success             string
	Error               string
	Subscriptions       []adminSubRow
	Categories          []Category
	PollIntervalMinutes int
	ItemCap             int
	APIKeyConfigured    bool
}

type createSubscriptionBody struct {
	FeedURL      string `json:"feed_url"`
	Title        string `json:"title"`
	CategoryID   *int64 `json:"category_id"`
	CategoryName string `json:"category_name"`
}

type updateSubscriptionBody struct {
	CategoryID    *int64 `json:"category_id"`
	CategoryName  string `json:"category_name"`
	ClearCategory bool   `json:"clear_category"`
}

type createCategoryBody struct {
	Name string `json:"name"`
}

type updateSettingsBody struct {
	PollIntervalMinutes *int `json:"poll_interval_minutes"`
}

type discoverBody struct {
	BaseURL string `json:"base_url"`
}

type importSummary struct {
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Failed   []string `json:"failed"`
}

type subscriptionView struct {
	ID            int64   `json:"id"`
	FeedURL       string  `json:"feed_url"`
	Title         string  `json:"title"`
	SiteURL       *string `json:"site_url,omitempty"`
	FaviconURL    *string `json:"favicon_url,omitempty"`
	CategoryID    *int64  `json:"category_id,omitempty"`
	ETag          *string `json:"etag,omitempty"`
	LastModified  *string `json:"last_modified,omitempty"`
	LastFetchedAt *string `json:"last_fetched_at,omitempty"`
	LastError     *string `json:"last_error,omitempty"`
	AddedAt       string  `json:"added_at"`
}
