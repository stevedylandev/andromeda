package main

import (
	"database/sql"
	"testing"

	sharedsqlite "github.com/stevedylandev/andromeda/crates-go/sqlite"
)

func openBookmarksTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sharedsqlite.Open("file:bookmarks-test?mode=memory&cache=shared", schema)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestCategoriesAndBookmarkCRUD(t *testing.T) {
	db := openBookmarksTestDB(t)
	defer db.Close()
	cat, err := createCategory(db, "Reading")
	if err != nil {
		t.Fatal(err)
	}
	if cat.ShortID == "" || cat.Position != 1 {
		t.Fatalf("category %#v", cat)
	}
	if got, err := getCategoryByName(db, " Reading "); err != nil || got == nil || got.ID != cat.ID {
		t.Fatalf("category by name %#v err %v", got, err)
	}
	cats, err := listCategories(db)
	if err != nil || len(cats) != 1 {
		t.Fatalf("cats %#v err %v", cats, err)
	}

	fav := "https://example.com/favicon.ico"
	link, err := createLink(db, "Example", "https://example.com", &fav, cat.ID)
	if err != nil {
		t.Fatal(err)
	}
	if link.ShortID == "" || link.FaviconURL == nil || *link.FaviconURL != fav {
		t.Fatalf("link %#v", link)
	}
	dup, err := createLink(db, "Example 2", "https://example.com", nil, cat.ID)
	if err != nil {
		t.Fatal(err)
	}
	if dup.ID == link.ID {
		t.Fatal("duplicate URL should create a distinct bookmark")
	}
	links, err := listLinks(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 2 {
		t.Fatalf("links %#v", links)
	}
	missingFav, err := listLinksMissingFavicon(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(missingFav) != 1 || missingFav[0].ID != dup.ID {
		t.Fatalf("missing fav %#v", missingFav)
	}
	if err := updateLinkFavicon(db, dup.ID, &fav); err != nil {
		t.Fatal(err)
	}
	missingFav, err = listLinksMissingFavicon(db)
	if err != nil || len(missingFav) != 0 {
		t.Fatalf("missing after update %#v err %v", missingFav, err)
	}

	deleted, err := deleteLinkByShortID(db, link.ShortID)
	if err != nil || !deleted {
		t.Fatalf("delete link %v err %v", deleted, err)
	}
	deleted, err = deleteLinkByShortID(db, "missing")
	if err != nil || deleted {
		t.Fatalf("delete missing link %v err %v", deleted, err)
	}
	deleted, err = deleteCategoryByShortID(db, cat.ShortID)
	if err != nil || !deleted {
		t.Fatalf("delete category %v err %v", deleted, err)
	}
}
