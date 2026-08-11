package main

import (
	"encoding/csv"
	"net/http"
	"strings"
	"time"

	"github.com/stevedylandev/andromeda/pkg/web"
)

// exportHandler streams records as CSV. With ?habit=<short_id> it exports a
// single habit; otherwise every habit's records. Rows are chronological.
func (a *App) exportHandler(w http.ResponseWriter, r *http.Request) {
	habitShortID := strings.TrimSpace(r.URL.Query().Get("habit"))

	filename := "habbits-export.csv"
	if habitShortID != "" {
		habit, err := getHabitByShortID(a.DB, habitShortID)
		if err != nil {
			a.Log.Error("export get habit", "err", err)
			web.RedirectWithError(w, r, "/settings", "Failed to export")
			return
		}
		if habit == nil {
			http.NotFound(w, r)
			return
		}
		filename = "habbits-" + slugify(habit.Name) + ".csv"
	}

	records, err := recordsForExport(a.DB, habitShortID)
	if err != nil {
		a.Log.Error("export records", "err", err)
		web.RedirectWithError(w, r, "/settings", "Failed to export")
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"recorded_at", "habit", "value_type", "unit", "value"})
	for _, rec := range records {
		_ = cw.Write([]string{
			time.Unix(rec.RecordedAt, 0).Local().Format(time.RFC3339),
			rec.HabitName,
			rec.ValueType,
			rec.Unit,
			rec.Value,
		})
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		a.Log.Error("export csv write", "err", err)
	}
}

// slugify makes a filesystem-safe token from a habit name for the download
// filename, e.g. "Sleep hours" -> "sleep-hours".
func slugify(s string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "habit"
	}
	return out
}
