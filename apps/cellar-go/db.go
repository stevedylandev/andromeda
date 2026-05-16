package main

import (
	"database/sql"
	"errors"

	"github.com/stevedylandev/andromeda/crates-go/auth"
	_ "modernc.org/sqlite"
)

const cellarSchema = `
CREATE TABLE IF NOT EXISTS wines (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    short_id        TEXT NOT NULL UNIQUE,
    name            TEXT NOT NULL,
    origin          TEXT NOT NULL,
    grape           TEXT NOT NULL,
    notes           TEXT NOT NULL,
    image           BLOB,
    image_mime      TEXT,
    sweetness       INTEGER NOT NULL CHECK(sweetness BETWEEN 1 AND 5),
    acidity         INTEGER NOT NULL CHECK(acidity BETWEEN 1 AND 5),
    tannin          INTEGER NOT NULL CHECK(tannin BETWEEN 1 AND 5),
    alcohol         INTEGER NOT NULL CHECK(alcohol BETWEEN 1 AND 5),
    body            INTEGER NOT NULL CHECK(body BETWEEN 1 AND 5),
    clarity         INTEGER NOT NULL DEFAULT 3,
    color_intensity INTEGER NOT NULL DEFAULT 3,
    aroma_intensity INTEGER NOT NULL DEFAULT 3,
    nose_complexity INTEGER NOT NULL DEFAULT 3,
    background      TEXT NOT NULL DEFAULT '',
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    wishlist        INTEGER NOT NULL DEFAULT 0
);
`

const wineCols = `id, short_id, name, origin, grape, notes, (image IS NOT NULL) AS has_image, COALESCE(image_mime, ''), sweetness, acidity, tannin, alcohol, body, clarity, color_intensity, aroma_intensity, nose_complexity, background, created_at, wishlist`

func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if _, err := db.Exec(cellarSchema); err != nil {
		return nil, err
	}
	return db, nil
}

func scanWine(s interface{ Scan(...any) error }) (*Wine, error) {
	var w Wine
	var hasImage int
	err := s.Scan(&w.ID, &w.ShortID, &w.Name, &w.Origin, &w.Grape, &w.Notes,
		&hasImage, &w.ImageMime,
		&w.Sweetness, &w.Acidity, &w.Tannin, &w.Alcohol, &w.Body,
		&w.Clarity, &w.ColorIntensity, &w.AromaIntensity, &w.NoseComplexity,
		&w.Background, &w.CreatedAt, &w.Wishlist)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	w.HasImage = hasImage != 0
	return &w, nil
}

func createWine(db *sql.DB, in WineInput, wishlist bool) (*Wine, error) {
	shortID, err := auth.GenerateShortID(10)
	if err != nil {
		return nil, err
	}
	res, err := db.Exec(
		`INSERT INTO wines (short_id, name, origin, grape, notes, sweetness, acidity, tannin, alcohol, body, clarity, color_intensity, aroma_intensity, nose_complexity, background, wishlist)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		shortID, in.Name, in.Origin, in.Grape, in.Notes,
		in.Sweetness, in.Acidity, in.Tannin, in.Alcohol, in.Body,
		in.Clarity, in.ColorIntensity, in.AromaIntensity, in.NoseComplexity,
		in.Background, boolToInt(wishlist),
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return scanWine(db.QueryRow(`SELECT `+wineCols+` FROM wines WHERE id = ?`, id))
}

func getCellarWines(db *sql.DB) ([]Wine, error) {
	rows, err := db.Query(`SELECT ` + wineCols + ` FROM wines WHERE wishlist = 0 ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Wine{}
	for rows.Next() {
		w, err := scanWine(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *w)
	}
	return out, rows.Err()
}

func getWishlistWines(db *sql.DB) ([]Wine, error) {
	rows, err := db.Query(`SELECT ` + wineCols + ` FROM wines WHERE wishlist = 1 ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Wine{}
	for rows.Next() {
		w, err := scanWine(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *w)
	}
	return out, rows.Err()
}

func getWineByShortID(db *sql.DB, shortID string) (*Wine, error) {
	return scanWine(db.QueryRow(`SELECT `+wineCols+` FROM wines WHERE short_id = ?`, shortID))
}

func getWineImage(db *sql.DB, shortID string) ([]byte, string, error) {
	var img []byte
	var mime string
	err := db.QueryRow(`SELECT image, COALESCE(image_mime, '') FROM wines WHERE short_id = ? AND image IS NOT NULL`, shortID).Scan(&img, &mime)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	return img, mime, nil
}

func updateWine(db *sql.DB, shortID string, in WineInput) (*Wine, error) {
	res, err := db.Exec(
		`UPDATE wines SET name = ?, origin = ?, grape = ?, notes = ?,
		 sweetness = ?, acidity = ?, tannin = ?, alcohol = ?, body = ?,
		 clarity = ?, color_intensity = ?, aroma_intensity = ?, nose_complexity = ?,
		 background = ? WHERE short_id = ?`,
		in.Name, in.Origin, in.Grape, in.Notes,
		in.Sweetness, in.Acidity, in.Tannin, in.Alcohol, in.Body,
		in.Clarity, in.ColorIntensity, in.AromaIntensity, in.NoseComplexity,
		in.Background, shortID,
	)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, nil
	}
	return getWineByShortID(db, shortID)
}

func updateWishlistWine(db *sql.DB, shortID, name, origin, grape, notes, background string) (*Wine, error) {
	res, err := db.Exec(
		`UPDATE wines SET name = ?, origin = ?, grape = ?, notes = ?, background = ? WHERE short_id = ? AND wishlist = 1`,
		name, origin, grape, notes, background, shortID,
	)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, nil
	}
	return getWineByShortID(db, shortID)
}

func promoteWine(db *sql.DB, shortID string) (bool, error) {
	res, err := db.Exec(`UPDATE wines SET wishlist = 0 WHERE short_id = ? AND wishlist = 1`, shortID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func updateWineImage(db *sql.DB, shortID string, image []byte, mime string) error {
	_, err := db.Exec(`UPDATE wines SET image = ?, image_mime = ? WHERE short_id = ?`, image, mime, shortID)
	return err
}

func deleteWine(db *sql.DB, shortID string) error {
	_, err := db.Exec(`DELETE FROM wines WHERE short_id = ?`, shortID)
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
