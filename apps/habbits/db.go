package main

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/stevedylandev/andromeda/pkg/auth"
)

const habbitsSchema = `
CREATE TABLE IF NOT EXISTS habits (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    short_id    TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    value_type  TEXT NOT NULL CHECK(value_type IN ('int','float','bool','string')),
    unit        TEXT,
    description TEXT,
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS records (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    short_id    TEXT NOT NULL UNIQUE,
    habit_id    INTEGER NOT NULL REFERENCES habits(id) ON DELETE CASCADE,
    value       TEXT NOT NULL,
    recorded_at INTEGER NOT NULL,
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_records_habit ON records(habit_id, recorded_at DESC);
`

type Habit struct {
	ID          int64   `json:"id"`
	ShortID     string  `json:"short_id"`
	Name        string  `json:"name"`
	ValueType   string  `json:"value_type"`
	Unit        *string `json:"unit,omitempty"`
	Description *string `json:"description,omitempty"`
	CreatedAt   int64   `json:"created_at"`
	UpdatedAt   int64   `json:"updated_at"`
}

type Record struct {
	ID         int64  `json:"id"`
	ShortID    string `json:"short_id"`
	HabitID    int64  `json:"habit_id"`
	Value      string `json:"value"`
	RecordedAt int64  `json:"recorded_at"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
	// Populated by joined queries; not stored on the records table.
	HabitShortID string `json:"habit_short_id,omitempty"`
	HabitName    string `json:"habit_name,omitempty"`
	ValueType    string `json:"value_type,omitempty"`
	Unit         string `json:"unit,omitempty"`
}

const habitCols = `id, short_id, name, value_type, unit, description, created_at, updated_at`

func scanHabit(s interface{ Scan(...any) error }) (*Habit, error) {
	var h Habit
	var unit, desc sql.NullString
	err := s.Scan(&h.ID, &h.ShortID, &h.Name, &h.ValueType, &unit, &desc, &h.CreatedAt, &h.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if unit.Valid {
		v := unit.String
		h.Unit = &v
	}
	if desc.Valid {
		v := desc.String
		h.Description = &v
	}
	return &h, nil
}

// listHabits returns all habits ordered by name, each annotated with its record
// count via a LEFT JOIN so habits with zero records still appear.
func listHabits(db *sql.DB) ([]Habit, map[int64]int, error) {
	rows, err := db.Query(`
		SELECT ` + prefixCols("h", habitCols) + `, COUNT(r.id)
		FROM habits h
		LEFT JOIN records r ON r.habit_id = h.id
		GROUP BY h.id
		ORDER BY h.name COLLATE NOCASE ASC`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var out []Habit
	counts := map[int64]int{}
	for rows.Next() {
		var h Habit
		var unit, desc sql.NullString
		var count int
		if err := rows.Scan(&h.ID, &h.ShortID, &h.Name, &h.ValueType, &unit, &desc, &h.CreatedAt, &h.UpdatedAt, &count); err != nil {
			return nil, nil, err
		}
		if unit.Valid {
			v := unit.String
			h.Unit = &v
		}
		if desc.Valid {
			v := desc.String
			h.Description = &v
		}
		out = append(out, h)
		counts[h.ID] = count
	}
	return out, counts, rows.Err()
}

func getHabitByShortID(db *sql.DB, shortID string) (*Habit, error) {
	return scanHabit(db.QueryRow(`SELECT `+habitCols+` FROM habits WHERE short_id = ?`, shortID))
}

func insertHabit(db *sql.DB, name, valueType, unit, description string) (int64, error) {
	shortID, err := auth.GenerateShortID(10)
	if err != nil {
		return 0, err
	}
	now := time.Now().UTC().Unix()
	res, err := db.Exec(
		`INSERT INTO habits (short_id, name, value_type, unit, description, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		shortID, name, valueType, nullify(unit), nullify(description), now, now,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func updateHabit(db *sql.DB, shortID, name, valueType, unit, description string) error {
	_, err := db.Exec(
		`UPDATE habits SET name = ?, value_type = ?, unit = ?, description = ?, updated_at = ?
		 WHERE short_id = ?`,
		name, valueType, nullify(unit), nullify(description), time.Now().UTC().Unix(), shortID,
	)
	return err
}

func deleteHabitByShortID(db *sql.DB, shortID string) error {
	_, err := db.Exec(`DELETE FROM habits WHERE short_id = ?`, shortID)
	return err
}

const recordJoin = `
	SELECT r.id, r.short_id, r.habit_id, r.value, r.recorded_at, r.created_at, r.updated_at,
	       h.short_id, h.name, h.value_type, IFNULL(h.unit,'')
	FROM records r JOIN habits h ON h.id = r.habit_id`

func scanRecordJoined(s interface{ Scan(...any) error }) (*Record, error) {
	var r Record
	err := s.Scan(&r.ID, &r.ShortID, &r.HabitID, &r.Value, &r.RecordedAt, &r.CreatedAt, &r.UpdatedAt,
		&r.HabitShortID, &r.HabitName, &r.ValueType, &r.Unit)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func queryRecords(db *sql.DB, where string, args ...any) ([]Record, error) {
	rows, err := db.Query(recordJoin+" "+where, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Record
	for rows.Next() {
		rec, err := scanRecordJoined(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *rec)
	}
	return out, rows.Err()
}

func listRecords(db *sql.DB, limit int) ([]Record, error) {
	return queryRecords(db, `ORDER BY r.recorded_at DESC, r.id DESC LIMIT ?`, limit)
}

func listRecordsForHabit(db *sql.DB, habitID int64) ([]Record, error) {
	return queryRecords(db, `WHERE r.habit_id = ? ORDER BY r.recorded_at DESC, r.id DESC`, habitID)
}

func getRecordByShortID(db *sql.DB, shortID string) (*Record, error) {
	return scanRecordJoined(db.QueryRow(recordJoin+` WHERE r.short_id = ?`, shortID))
}

// recordsForExport returns records in chronological order for CSV export. When
// habitShortID is non-empty, only that habit's records are returned.
func recordsForExport(db *sql.DB, habitShortID string) ([]Record, error) {
	if habitShortID != "" {
		return queryRecords(db, `WHERE h.short_id = ? ORDER BY r.recorded_at ASC, r.id ASC`, habitShortID)
	}
	return queryRecords(db, `ORDER BY r.recorded_at ASC, r.id ASC`)
}

func insertRecord(db *sql.DB, habitID int64, value string, recordedAt int64) (int64, error) {
	shortID, err := auth.GenerateShortID(10)
	if err != nil {
		return 0, err
	}
	now := time.Now().UTC().Unix()
	res, err := db.Exec(
		`INSERT INTO records (short_id, habit_id, value, recorded_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		shortID, habitID, value, recordedAt, now, now,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func updateRecord(db *sql.DB, shortID, value string, recordedAt int64) error {
	_, err := db.Exec(
		`UPDATE records SET value = ?, recorded_at = ?, updated_at = ? WHERE short_id = ?`,
		value, recordedAt, time.Now().UTC().Unix(), shortID,
	)
	return err
}

func deleteRecordByShortID(db *sql.DB, shortID string) error {
	_, err := db.Exec(`DELETE FROM records WHERE short_id = ?`, shortID)
	return err
}

// nullify returns nil for empty/whitespace strings so they store as SQL NULL.
func nullify(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

// prefixCols rewrites a comma column list like "id, short_id" into
// "h.id, h.short_id" for use in JOIN selects.
func prefixCols(alias, cols string) string {
	parts := strings.Split(cols, ",")
	for i, p := range parts {
		parts[i] = " " + alias + "." + strings.TrimSpace(p)
	}
	return strings.Join(parts, ",")
}
