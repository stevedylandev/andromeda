package main

import (
	"net/http"
	"strings"
	"time"

	"github.com/stevedylandev/andromeda/pkg/auth"
	"github.com/stevedylandev/andromeda/pkg/web"
)

const (
	dateLayout  = "2006-01-02"
	timeLayout  = "15:04"
	inputLayout = "2006-01-02T15:04" // datetime-local
)

func habitToRow(h Habit, count int) habitRow {
	row := habitRow{ShortID: h.ShortID, Name: h.Name, ValueType: h.ValueType, RecordCount: count}
	if h.Unit != nil {
		row.Unit = *h.Unit
	}
	if h.Description != nil {
		row.Description = *h.Description
	}
	return row
}

func recordToRow(r Record) recordRow {
	t := time.Unix(r.RecordedAt, 0).Local()
	return recordRow{
		ShortID:         r.ShortID,
		HabitShortID:    r.HabitShortID,
		HabitName:       r.HabitName,
		ValueType:       r.ValueType,
		Value:           r.Value,
		Unit:            r.Unit,
		Date:            t.Format(dateLayout),
		TimeDisplay:     t.Format(timeLayout),
		RecordedAtInput: t.Format(inputLayout),
	}
}

// groupByDay buckets records (ordered recorded_at DESC) into day groups,
// preserving encounter order so records stay chronological within a day.
func groupByDay(recs []Record) []habitDay {
	var days []habitDay
	idx := map[string]int{}
	for _, r := range recs {
		row := recordToRow(r)
		i, ok := idx[row.Date]
		if !ok {
			i = len(days)
			idx[row.Date] = i
			days = append(days, habitDay{Date: row.Date})
		}
		days[i].Records = append(days[i].Records, row)
	}
	return days
}

// parseRecordedAt converts a datetime-local form value into unix seconds. An
// empty value defaults to now, matching the "auto-injected at time of entry"
// behaviour while still allowing a backdated entry from the UI.
func parseRecordedAt(raw string) (int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Now().Unix(), nil
	}
	t, err := time.ParseInLocation(inputLayout, raw, time.Local)
	if err != nil {
		return 0, err
	}
	return t.Unix(), nil
}

// --- auth ---

func (a *App) loginGetHandler(w http.ResponseWriter, r *http.Request) {
	web.Render(a.Templates, w, "login.html", loginPageData{Error: r.URL.Query().Get("error")}, a.Log)
}

func (a *App) loginPostHandler(w http.ResponseWriter, r *http.Request) {
	if a.AdminPassword == "" {
		web.RedirectWithError(w, r, "/login", "No admin password configured")
		return
	}
	if err := r.ParseForm(); err != nil {
		web.RedirectWithError(w, r, "/login", "Bad request")
		return
	}
	if !auth.VerifyPassword(r.FormValue("password"), a.AdminPassword) {
		web.RedirectWithError(w, r, "/login", "Invalid password")
		return
	}
	token, err := a.Sessions.Create()
	if err != nil {
		a.Log.Error("create session failed", "err", err)
		web.RedirectWithError(w, r, "/login", "Session error")
		return
	}
	a.Sessions.PruneExpired()
	http.SetCookie(w, a.Sessions.SessionCookie(token))
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) logoutHandler(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(a.Sessions.CookieName); err == nil && c.Value != "" {
		a.Sessions.Delete(c.Value)
	}
	http.SetCookie(w, a.Sessions.ClearCookie())
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// --- dashboard ---

func (a *App) dashboardHandler(w http.ResponseWriter, r *http.Request) {
	habits, counts, err := listHabits(a.DB)
	if err != nil {
		a.Log.Error("list habits", "err", err)
	}
	habitRows := make([]habitRow, 0, len(habits))
	for _, h := range habits {
		habitRows = append(habitRows, habitToRow(h, counts[h.ID]))
	}

	records, err := listRecords(a.DB, 200)
	if err != nil {
		a.Log.Error("list records", "err", err)
	}

	web.Render(a.Templates, w, "index.html", dashboardData{
		Success: r.URL.Query().Get("success"),
		Error:   r.URL.Query().Get("error"),
		Habits:  habitRows,
		Days:    groupByDay(records),
	}, a.Log)
}

// --- habits ---

func (a *App) newHabitHandler(w http.ResponseWriter, r *http.Request) {
	web.Render(a.Templates, w, "new.html", newHabitPageData{
		Error:      r.URL.Query().Get("error"),
		ValueTypes: valueTypes,
	}, a.Log)
}

func (a *App) settingsHandler(w http.ResponseWriter, r *http.Request) {
	habits, counts, err := listHabits(a.DB)
	if err != nil {
		a.Log.Error("list habits", "err", err)
	}
	habitRows := make([]habitRow, 0, len(habits))
	for _, h := range habits {
		habitRows = append(habitRows, habitToRow(h, counts[h.ID]))
	}
	web.Render(a.Templates, w, "settings.html", settingsPageData{
		Success:    r.URL.Query().Get("success"),
		Error:      r.URL.Query().Get("error"),
		ValueTypes: valueTypes,
		Habits:     habitRows,
	}, a.Log)
}

func (a *App) createHabitHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		web.RedirectWithError(w, r, "/new", "Bad request")
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	valueType := strings.TrimSpace(r.FormValue("value_type"))
	unit := strings.TrimSpace(r.FormValue("unit"))
	description := strings.TrimSpace(r.FormValue("description"))
	if name == "" {
		web.RedirectWithError(w, r, "/new", "Habit name is required")
		return
	}
	if !validValueType(valueType) {
		web.RedirectWithError(w, r, "/new", "Invalid value type")
		return
	}
	if _, err := insertHabit(a.DB, name, valueType, unit, description); err != nil {
		a.Log.Error("insert habit", "err", err)
		web.RedirectWithError(w, r, "/new", "Failed to create habit")
		return
	}
	web.RedirectWithSuccess(w, r, "/", "Habit created")
}

func (a *App) habitDetailHandler(w http.ResponseWriter, r *http.Request) {
	habit, err := getHabitByShortID(a.DB, r.PathValue("short_id"))
	if err != nil {
		a.Log.Error("get habit", "err", err)
		web.RedirectWithError(w, r, "/", "Failed to load habit")
		return
	}
	if habit == nil {
		http.NotFound(w, r)
		return
	}
	records, err := listRecordsForHabit(a.DB, habit.ID)
	if err != nil {
		a.Log.Error("list habit records", "err", err)
	}
	web.Render(a.Templates, w, "habit.html", habitPageData{
		Success: r.URL.Query().Get("success"),
		Error:   r.URL.Query().Get("error"),
		Habit:   habitToRow(*habit, len(records)),
		Days:    groupByDay(records),
	}, a.Log)
}

func (a *App) updateHabitHandler(w http.ResponseWriter, r *http.Request) {
	shortID := r.PathValue("short_id")
	target := "/settings"
	if err := r.ParseForm(); err != nil {
		web.RedirectWithError(w, r, target, "Bad request")
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	valueType := strings.TrimSpace(r.FormValue("value_type"))
	if name == "" {
		web.RedirectWithError(w, r, target, "Habit name is required")
		return
	}
	if !validValueType(valueType) {
		web.RedirectWithError(w, r, target, "Invalid value type")
		return
	}
	if err := updateHabit(a.DB, shortID, name, valueType,
		strings.TrimSpace(r.FormValue("unit")), strings.TrimSpace(r.FormValue("description"))); err != nil {
		a.Log.Error("update habit", "err", err)
		web.RedirectWithError(w, r, target, "Failed to update habit")
		return
	}
	web.RedirectWithSuccess(w, r, target, "Habit updated")
}

func (a *App) deleteHabitHandler(w http.ResponseWriter, r *http.Request) {
	target := "/settings"
	if ref := r.FormValue("return_to"); ref != "" {
		target = ref
	}
	if err := deleteHabitByShortID(a.DB, r.PathValue("short_id")); err != nil {
		a.Log.Error("delete habit", "err", err)
		web.RedirectWithError(w, r, target, "Failed to delete habit")
		return
	}
	web.RedirectWithSuccess(w, r, target, "Habit deleted")
}

// --- records ---

func (a *App) createRecordHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		web.RedirectWithError(w, r, "/", "Bad request")
		return
	}
	// Where to return to: the referring page (dashboard or habit detail).
	target := "/"
	if ref := r.FormValue("return_to"); ref != "" {
		target = ref
	}

	habit, err := getHabitByShortID(a.DB, strings.TrimSpace(r.FormValue("habit")))
	if err != nil {
		a.Log.Error("get habit for record", "err", err)
		web.RedirectWithError(w, r, target, "Failed to load habit")
		return
	}
	if habit == nil {
		web.RedirectWithError(w, r, target, "Select a habit")
		return
	}
	value, err := normalizeValue(habit.ValueType, r.FormValue("value"))
	if err != nil {
		web.RedirectWithError(w, r, target, err.Error())
		return
	}
	recordedAt, err := parseRecordedAt(r.FormValue("recorded_at"))
	if err != nil {
		web.RedirectWithError(w, r, target, "Invalid date/time")
		return
	}
	if _, err := insertRecord(a.DB, habit.ID, value, recordedAt); err != nil {
		a.Log.Error("insert record", "err", err)
		web.RedirectWithError(w, r, target, "Failed to add record")
		return
	}
	web.RedirectWithSuccess(w, r, target, "Record added")
}

func (a *App) updateRecordHandler(w http.ResponseWriter, r *http.Request) {
	shortID := r.PathValue("short_id")
	if err := r.ParseForm(); err != nil {
		web.RedirectWithError(w, r, "/", "Bad request")
		return
	}
	rec, err := getRecordByShortID(a.DB, shortID)
	if err != nil {
		a.Log.Error("get record", "err", err)
		web.RedirectWithError(w, r, "/", "Failed to load record")
		return
	}
	if rec == nil {
		http.NotFound(w, r)
		return
	}
	target := "/habits/" + rec.HabitShortID
	if ref := r.FormValue("return_to"); ref != "" {
		target = ref
	}
	value, err := normalizeValue(rec.ValueType, r.FormValue("value"))
	if err != nil {
		web.RedirectWithError(w, r, target, err.Error())
		return
	}
	recordedAt, err := parseRecordedAt(r.FormValue("recorded_at"))
	if err != nil {
		web.RedirectWithError(w, r, target, "Invalid date/time")
		return
	}
	if err := updateRecord(a.DB, shortID, value, recordedAt); err != nil {
		a.Log.Error("update record", "err", err)
		web.RedirectWithError(w, r, target, "Failed to update record")
		return
	}
	web.RedirectWithSuccess(w, r, target, "Record updated")
}

func (a *App) deleteRecordHandler(w http.ResponseWriter, r *http.Request) {
	target := "/"
	if ref := r.FormValue("return_to"); ref != "" {
		target = ref
	}
	if err := deleteRecordByShortID(a.DB, r.PathValue("short_id")); err != nil {
		a.Log.Error("delete record", "err", err)
		web.RedirectWithError(w, r, target, "Failed to delete record")
		return
	}
	web.RedirectWithSuccess(w, r, target, "Record deleted")
}
