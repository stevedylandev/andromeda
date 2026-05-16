package main

import (
	"database/sql"
	"embed"
	"html/template"
	"log/slog"
)

//go:embed templates/*.html static/* static/fonts/* assets/* assets/fonts/*
var appFS embed.FS

type App struct {
	DB           *sql.DB
	Log          *slog.Logger
	Templates    *template.Template
	Password     string
	APIKey       string
	CookieSecure bool
}

type Note struct {
	ID        int64  `json:"id"`
	ShortID   string `json:"short_id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type NoteInput struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

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
