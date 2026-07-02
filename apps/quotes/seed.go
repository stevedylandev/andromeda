package main

import (
	_ "embed"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/stevedylandev/andromeda/pkg/auth"
	"github.com/stevedylandev/andromeda/pkg/sqlite"
)

const classicListPath = "classic_authors.txt"

// embeddedClassicList is baked into the binary so the seed command works inside
// a container (or anywhere the txt file is absent) without needing the file on
// disk. An on-disk classic_authors.txt still takes precedence, so local edits
// apply without a rebuild.
//
//go:embed classic_authors.txt
var embeddedClassicList string

func parseClassicList(data string) []string {
	var out []string
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out
}

// loadClassicList reads the match list from disk, falling back to the embedded
// copy when the file is absent. Skips blank lines and # comments.
func loadClassicList(path string) ([]string, bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return parseClassicList(embeddedClassicList), true, nil
	}
	if err != nil {
		return nil, false, err
	}
	return parseClassicList(string(data)), false, nil
}

// matchesClassic reports whether the author column contains any list entry as a
// case-sensitive substring.
func matchesClassic(author string, list []string) bool {
	for _, name := range list {
		if strings.Contains(author, name) {
			return true
		}
	}
	return false
}

// splitAttribution splits a CSV author field of the form "Author, Book Title"
// at the first comma into author and source. No comma -> empty source.
func splitAttribution(field string) (author, source string) {
	if i := strings.Index(field, ","); i >= 0 {
		return strings.TrimSpace(field[:i]), strings.TrimSpace(field[i+1:])
	}
	return strings.TrimSpace(field), ""
}

func listSource(embedded bool) string {
	if embedded {
		return "embedded"
	}
	return classicListPath
}

func dedupKey(text, author string) string {
	return text + "\x00" + author
}

// runSeed imports classic-literature quotes from csvPath into the database at
// dbPath. It is idempotent: quotes already present (matched on text+author) are
// skipped, so re-running after editing classic_authors.txt only adds new rows.
func runSeed(logger *slog.Logger, dbPath, csvPath string) error {
	list, embedded, err := loadClassicList(classicListPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", classicListPath, err)
	}
	logger.Info("loaded classic list", "entries", len(list), "source", listSource(embedded))

	db, err := sqlite.Open(dbPath, quotesSchema)
	if err != nil {
		return err
	}
	defer db.Close()

	// Preload existing (text, author) pairs so re-seeds stay idempotent.
	seen := map[string]struct{}{}
	rows, err := db.Query(`SELECT text, author FROM quotes`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var t, a string
		if err := rows.Scan(&t, &a); err != nil {
			rows.Close()
			return err
		}
		seen[dedupKey(t, a)] = struct{}{}
	}
	rows.Close()

	f, err := os.Open(csvPath)
	if err != nil {
		return err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.FieldsPerRecord = -1
	if _, err := reader.Read(); err != nil { // skip header
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(
		`INSERT INTO quotes (short_id, text, author, source, added_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	now := time.Now().UTC().Unix()
	var scanned, inserted, skipped int
	for {
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.Warn("skipping malformed row", "err", err)
			continue
		}
		if len(rec) < 2 {
			continue
		}
		scanned++
		text := strings.TrimSpace(rec[0])
		rawAuthor := rec[1]
		if text == "" || !matchesClassic(rawAuthor, list) {
			continue
		}
		author, source := splitAttribution(rawAuthor)
		if author == "" {
			continue
		}
		key := dedupKey(text, author)
		if _, ok := seen[key]; ok {
			skipped++
			continue
		}
		shortID, err := auth.GenerateShortID(10)
		if err != nil {
			tx.Rollback()
			return err
		}
		var src any
		if source != "" {
			src = source
		}
		if _, err := stmt.Exec(shortID, text, author, src, now, now); err != nil {
			tx.Rollback()
			return err
		}
		seen[key] = struct{}{}
		inserted++
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	total, _ := countQuotes(db)
	logger.Info("seed complete", "scanned", scanned, "inserted", inserted, "skipped_duplicates", skipped, "total_in_db", total)
	return nil
}
