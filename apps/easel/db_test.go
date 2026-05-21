package main

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	sharedsqlite "github.com/stevedylandev/andromeda/crates-go/sqlite"
)

func openEaselTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sharedsqlite.Open("file:easel-test?mode=memory&cache=shared", easelSchema)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func daily(date string, id int64) *DailyArtwork {
	return &DailyArtwork{Date: date, ArtworkID: id, Title: "Title", ImageID: "img", FetchedAt: "now"}
}

func TestDailyArtworkDBHelpers(t *testing.T) {
	db := openEaselTestDB(t)
	defer db.Close()
	ok, err := insertDaily(db, daily("2024-01-01", 1))
	if err != nil || !ok {
		t.Fatalf("insert %v err %v", ok, err)
	}
	ok, err = insertDaily(db, daily("2024-01-01", 2))
	if err != nil || ok {
		t.Fatalf("duplicate %v err %v", ok, err)
	}
	if got, err := getDaily(db, "2024-01-01"); err != nil || got == nil || got.ArtworkID != 1 {
		t.Fatalf("get %#v err %v", got, err)
	}
	if exists, err := artworkIDExists(db, 1); err != nil || !exists {
		t.Fatalf("exists %v err %v", exists, err)
	}
	if exists, err := artworkIDExists(db, 99); err != nil || exists {
		t.Fatalf("missing exists %v err %v", exists, err)
	}
	if _, err := insertDaily(db, daily("2024-01-03", 3)); err != nil {
		t.Fatal(err)
	}
	list, err := listDaily(db, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].Date != "2024-01-03" {
		t.Fatalf("list order %#v", list)
	}
	missing, err := missingDates(db, []string{"2024-01-01", "2024-01-02", "2024-01-03"})
	if err != nil {
		t.Fatal(err)
	}
	if len(missing) != 1 || missing[0] != "2024-01-02" {
		t.Fatalf("missing %#v", missing)
	}
}

func TestAICAndSchedulerHelpers(t *testing.T) {
	params := buildAICParams([]string{"Painting", "Print"}, []string{"war"})
	if !strings.Contains(params, "painting") || strings.Contains(params, "Painting") {
		t.Fatalf("classifications not lowercased: %s", params)
	}
	if lower("AbC!") != "abc!" {
		t.Fatal("lower failed")
	}
	title, img, artist := "Mona", "image-id", "Artist"
	d := rawToDaily(&rawArtwork{ID: 42, Title: &title, ImageID: &img, ArtistTitle: &artist}, "2024-01-01", "fetched")
	if d == nil || d.Title != title || d.ImageID != img || !d.ArtistTitle.Valid {
		t.Fatalf("daily %#v", d)
	}
	if rawToDaily(&rawArtwork{ID: 1}, "2024-01-01", "now") != nil {
		t.Fatal("missing image should return nil")
	}
	if rawToDaily(&rawArtwork{ID: 1, ImageID: &img}, "2024-01-01", "now").Title != "Untitled" {
		t.Fatal("expected untitled fallback")
	}

	app := &App{TZ: time.UTC}
	dates := app.pastNDates(3)
	if len(dates) != 3 {
		t.Fatalf("dates %#v", dates)
	}
	if today := time.Now().UTC().Format("2006-01-02"); dates[0] == today {
		t.Fatal("past dates included today")
	}
	if _, ok := parseDate("2024-02-29"); !ok {
		t.Fatal("valid date rejected")
	}
	if _, ok := parseDate("2024-02-30"); ok {
		t.Fatal("invalid date accepted")
	}
}
