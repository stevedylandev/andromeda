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

// valueTypes is the fixed set of value types a habit may declare. Kept in sync
// with the CHECK constraint in habbitsSchema and the normalizeValue switch.
var valueTypes = []string{"int", "float", "bool", "string"}

// habitRow is the view model for a habit in list/detail templates.
type habitRow struct {
	ShortID     string
	Name        string
	ValueType   string
	Unit        string
	Description string
	RecordCount int
}

// recordRow is the view model for a record. Date is the grouping key
// (YYYY-MM-DD); TimeDisplay is the time-of-day shown within a day group;
// RecordedAtInput is the datetime-local value for edit forms.
type recordRow struct {
	ShortID         string
	HabitShortID    string
	HabitName       string
	ValueType       string
	Value           string
	Unit            string
	Date            string
	TimeDisplay     string
	RecordedAtInput string
}

// habitDay groups records by day. Used for both the dashboard (all habits,
// records interleaved chronologically) and the habit detail page.
type habitDay struct {
	Date    string
	Records []recordRow
}

type loginPageData struct {
	Error string
}

type dashboardData struct {
	Success string
	Error   string
	Habits  []habitRow
	Days    []habitDay
}

type newHabitPageData struct {
	Error      string
	ValueTypes []string
}

type settingsPageData struct {
	Success    string
	Error      string
	ValueTypes []string
	Habits     []habitRow
}

type habitPageData struct {
	Success string
	Error   string
	Habit   habitRow
	Days    []habitDay
}
