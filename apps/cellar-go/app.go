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

type App struct {
	DB              *sql.DB
	Log             *slog.Logger
	Templates       *template.Template
	Sessions        *auth.Store
	AppPassword     string
	CookieSecure    bool
	AnthropicAPIKey string
	SiteURL         string
	SiteTitle       string
	SiteDescription string
}

type Wine struct {
	ID             int64  `json:"id"`
	ShortID        string `json:"short_id"`
	Name           string `json:"name"`
	Origin         string `json:"origin"`
	Grape          string `json:"grape"`
	Notes          string `json:"notes"`
	HasImage       bool   `json:"has_image"`
	ImageMime      string `json:"image_mime,omitempty"`
	Sweetness      int    `json:"sweetness"`
	Acidity        int    `json:"acidity"`
	Tannin         int    `json:"tannin"`
	Alcohol        int    `json:"alcohol"`
	Body           int    `json:"body"`
	Clarity        int    `json:"clarity"`
	ColorIntensity int    `json:"color_intensity"`
	AromaIntensity int    `json:"aroma_intensity"`
	NoseComplexity int    `json:"nose_complexity"`
	Background     string `json:"background"`
	CreatedAt      string `json:"created_at"`
	Wishlist       bool   `json:"wishlist"`
}

type WineInput struct {
	Name           string
	Origin         string
	Grape          string
	Notes          string
	Background     string
	Sweetness      int
	Acidity        int
	Tannin         int
	Alcohol        int
	Body           int
	Clarity        int
	ColorIntensity int
	AromaIntensity int
	NoseComplexity int
}

type wineWithSVG struct {
	Wine        Wine
	PentagonSVG template.HTML
}

type indexPageData struct {
	Wines []wineWithSVG
}

type loginPageData struct {
	Error string
	Next  string
}

type wineDetailPageData struct {
	Wine        Wine
	PentagonSVG template.HTML
	BarsSVG     template.HTML
}

type adminPageData struct {
	Wines []Wine
}

type wineFormPageData struct {
	Wine            *Wine
	Error           string
	HasAnthropicKey bool
}

type wishlistPageData struct {
	Wines   []Wine
	IsAdmin bool
}
