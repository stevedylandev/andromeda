package main

import (
	"database/sql"
	"testing"

	sharedsqlite "github.com/stevedylandev/andromeda/pkg/sqlite"
)

func openLibraryTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sharedsqlite.Open("file:library-test?mode=memory&cache=shared", booksSchema)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func sp(s string) *string { return &s }

func TestBookCRUDSearchSettingsAndGoogleHelpers(t *testing.T) {
	db := openLibraryTestDB(t)
	defer db.Close()
	id, err := insertBook(db, NewBook{GoogleID: sp("gid"), Title: "Dune", Authors: "Frank Herbert", ISBN: sp("123"), CoverURL: sp("https://cover"), Notes: sp("note"), Status: "want"})
	if err != nil {
		t.Fatal(err)
	}
	book, err := getBook(db, id)
	if err != nil || book == nil || book.Title != "Dune" || book.GoogleID == nil || *book.GoogleID != "gid" {
		t.Fatalf("book %#v err %v", book, err)
	}
	if err := updateBookStatus(db, id, "reading"); err != nil {
		t.Fatal(err)
	}
	note := "updated"
	if err := updateBookNotes(db, id, &note); err != nil {
		t.Fatal(err)
	}
	book, _ = getBook(db, id)
	if book.Status != "reading" || book.Notes == nil || *book.Notes != note {
		t.Fatalf("updated book %#v", book)
	}
	books, err := listBooks(db, "reading")
	if err != nil || len(books) != 1 || books[0].ID != id {
		t.Fatalf("filtered %#v err %v", books, err)
	}
	books, err = searchBooks(db, "herbert")
	if err != nil || len(books) != 1 {
		t.Fatalf("search %#v err %v", books, err)
	}
	if err := deleteBook(db, id); err != nil {
		t.Fatal(err)
	}
	if got, err := getBook(db, id); err != nil || got != nil {
		t.Fatalf("deleted %#v err %v", got, err)
	}

	if _, ok, err := getSetting(db, "missing"); err != nil || ok {
		t.Fatalf("missing setting ok=%v err=%v", ok, err)
	}
	if err := setSetting(db, "category_label.want", "Wishlist"); err != nil {
		t.Fatal(err)
	}
	labels, err := getCategoryLabels(db)
	if err != nil {
		t.Fatal(err)
	}
	if labels.Want != "Wishlist" || labels.Read == "" || labels.Reading == "" {
		t.Fatalf("labels %#v", labels)
	}

	if got := pickISBN([]identifier{{Kind: "ISBN_10", Identifier: "ten"}, {Kind: "ISBN_13", Identifier: "thirteen"}}); got != "thirteen" {
		t.Fatalf("isbn %q", got)
	}
	if !isISBNChars("123456789X") || isISBNChars("123ABC") {
		t.Fatal("isbn char validation failed")
	}
}
