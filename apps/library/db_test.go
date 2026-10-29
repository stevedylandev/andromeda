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

	if !isISBNChars("123456789X") || isISBNChars("123ABC") {
		t.Fatal("isbn char validation failed")
	}
}

func TestQuoteCRUDFilterAndCascade(t *testing.T) {
	db := openLibraryTestDB(t)
	defer db.Close()

	duneID, err := insertBook(db, NewBook{Title: "Dune", Authors: "Frank Herbert", Status: "read"})
	if err != nil {
		t.Fatal(err)
	}
	emmaID, err := insertBook(db, NewBook{Title: "Emma", Authors: "Jane Austen", Status: "read"})
	if err != nil {
		t.Fatal(err)
	}

	// Create (defaults to public).
	qid, err := insertQuote(db, NewQuote{BookID: duneID, Page: sp("12"), Quote: "Fear is the mind-killer."})
	if err != nil {
		t.Fatal(err)
	}
	q, err := getQuote(db, qid)
	if err != nil || q == nil || q.BookID != duneID || q.Quote != "Fear is the mind-killer." || !q.Public {
		t.Fatalf("get quote %#v err %v", q, err)
	}
	if q.Page == nil || *q.Page != "12" {
		t.Fatalf("page %#v", q.Page)
	}

	// Edit (all fields incl. book and page clearing).
	if err := updateQuote(db, qid, NewQuote{BookID: emmaID, Quote: "Changed quote."}); err != nil {
		t.Fatal(err)
	}
	q, _ = getQuote(db, qid)
	if q == nil || q.BookID != emmaID || q.Quote != "Changed quote." || q.Page != nil {
		t.Fatalf("updated quote %#v", q)
	}

	// Private quotes are excluded from public listing.
	private := false
	privID, err := insertQuote(db, NewQuote{BookID: emmaID, Quote: "secret quote.", Public: &private})
	if err != nil {
		t.Fatal(err)
	}
	if got, err := listQuotes(db, true); err != nil || len(got) != 1 || !got[0].Public {
		t.Fatalf("public list %#v err %v", got, err)
	}
	all, err := listQuotes(db, false)
	if err != nil || len(all) != 2 {
		t.Fatalf("all list %#v err %v", all, err)
	}
	// Joined metadata should come along.
	for _, qb := range all {
		if qb.BookTitle == "" || qb.BookAuthors == "" {
			t.Fatalf("quote with book missing join data: %#v", qb)
		}
	}

	// listQuotesForBook filters by book and respects publicOnly.
	got, err := listQuotesForBook(db, emmaID, false)
	if err != nil || len(got) != 2 {
		t.Fatalf("book quotes %#v err %v", got, err)
	}
	pub, err := listQuotesForBook(db, emmaID, true)
	if err != nil || len(pub) != 1 || pub[0].ID != qid {
		t.Fatalf("public book quotes %#v err %v", pub, err)
	}

	// Delete.
	if err := deleteQuote(db, privID); err != nil {
		t.Fatal(err)
	}
	if got, err := getQuote(db, privID); err != nil || got != nil {
		t.Fatalf("deleted quote %#v err %v", got, err)
	}

	// Deleting a book cascades its quotes.
	if err := deleteBook(db, emmaID); err != nil {
		t.Fatal(err)
	}
	if got, err := getQuote(db, qid); err != nil || got != nil {
		t.Fatalf("quote should cascade-delete with book: %#v err %v", got, err)
	}
}

func TestQuoteOrdering(t *testing.T) {
	db := openLibraryTestDB(t)
	defer db.Close()

	duneID, err := insertBook(db, NewBook{Title: "Dune", Authors: "Frank Herbert", Status: "read"})
	if err != nil {
		t.Fatal(err)
	}
	emmaID, err := insertBook(db, NewBook{Title: "Emma", Authors: "Jane Austen", Status: "read"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := insertQuote(db, NewQuote{BookID: emmaID, Quote: "Emma quote "}); err != nil {
		t.Fatal(err)
	}
	if _, err := insertQuote(db, NewQuote{BookID: duneID, Page: sp("1"), Quote: "dune p1 "}); err != nil {
		t.Fatal(err)
	}
	if _, err := insertQuote(db, NewQuote{BookID: duneID, Page: sp("2"), Quote: "dune p2 "}); err != nil {
		t.Fatal(err)
	}

	all, err := listQuotes(db, false)
	if err != nil || len(all) != 3 {
		t.Fatalf("list %#v err %v", all, err)
	}
	// Books grouped alphabetically: Dune before Emma.
	if all[0].BookTitle != "Dune" || all[2].BookTitle != "Emma" {
		t.Fatalf("group order wrong: %#v", all)
	}
	// Within a book, page order ascending.
	if all[0].Page == nil || *all[0].Page != "1" || all[1].Page == nil || *all[1].Page != "2" {
		t.Fatalf("page order wrong: %#v", all)
	}
}
