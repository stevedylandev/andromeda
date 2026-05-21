package main

import (
	"database/sql"
	"errors"
)

const easelSchema = `
CREATE TABLE IF NOT EXISTS daily_artworks (
    date              TEXT PRIMARY KEY,
    artwork_id        INTEGER NOT NULL,
    title             TEXT NOT NULL,
    artist_display    TEXT,
    artist_title      TEXT,
    date_display      TEXT,
    medium_display    TEXT,
    dimensions        TEXT,
    place_of_origin   TEXT,
    credit_line       TEXT,
    description       TEXT,
    short_description TEXT,
    image_id          TEXT NOT NULL,
    fetched_at        TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_daily_artworks_artwork_id ON daily_artworks(artwork_id);
`

type DailyArtwork struct {
	Date             string
	ArtworkID        int64
	Title            string
	ArtistDisplay    sql.NullString
	ArtistTitle      sql.NullString
	DateDisplay      sql.NullString
	MediumDisplay    sql.NullString
	Dimensions       sql.NullString
	PlaceOfOrigin    sql.NullString
	CreditLine       sql.NullString
	Description      sql.NullString
	ShortDescription sql.NullString
	ImageID          string
	FetchedAt        string
}

const dailyCols = `date, artwork_id, title, artist_display, artist_title, date_display, medium_display, dimensions, place_of_origin, credit_line, description, short_description, image_id, fetched_at`

func scanDaily(s interface{ Scan(...any) error }) (*DailyArtwork, error) {
	var d DailyArtwork
	err := s.Scan(&d.Date, &d.ArtworkID, &d.Title, &d.ArtistDisplay, &d.ArtistTitle,
		&d.DateDisplay, &d.MediumDisplay, &d.Dimensions, &d.PlaceOfOrigin, &d.CreditLine,
		&d.Description, &d.ShortDescription, &d.ImageID, &d.FetchedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func insertDaily(db *sql.DB, a *DailyArtwork) (bool, error) {
	res, err := db.Exec(
		`INSERT OR IGNORE INTO daily_artworks (`+dailyCols+`)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		a.Date, a.ArtworkID, a.Title, a.ArtistDisplay, a.ArtistTitle,
		a.DateDisplay, a.MediumDisplay, a.Dimensions, a.PlaceOfOrigin, a.CreditLine,
		a.Description, a.ShortDescription, a.ImageID, a.FetchedAt,
	)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func getDaily(db *sql.DB, date string) (*DailyArtwork, error) {
	return scanDaily(db.QueryRow(`SELECT `+dailyCols+` FROM daily_artworks WHERE date = ?`, date))
}

func listDaily(db *sql.DB, limit int) ([]DailyArtwork, error) {
	rows, err := db.Query(`SELECT `+dailyCols+` FROM daily_artworks ORDER BY date DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DailyArtwork
	for rows.Next() {
		d, err := scanDaily(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

func artworkIDExists(db *sql.DB, id int64) (bool, error) {
	var n int64
	err := db.QueryRow(`SELECT COUNT(*) FROM daily_artworks WHERE artwork_id = ?`, id).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func missingDates(db *sql.DB, dates []string) ([]string, error) {
	out := []string{}
	for _, d := range dates {
		var n int64
		if err := db.QueryRow(`SELECT COUNT(*) FROM daily_artworks WHERE date = ?`, d).Scan(&n); err != nil {
			return nil, err
		}
		if n == 0 {
			out = append(out, d)
		}
	}
	return out, nil
}
