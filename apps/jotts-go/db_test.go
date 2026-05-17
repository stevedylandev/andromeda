package main

import "testing"

func TestNoteCRUDAndOrdering(t *testing.T) {
	db, err := openDB("file:jotts-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	first, err := createNote(db, "first", "one")
	if err != nil {
		t.Fatal(err)
	}
	if first.ShortID == "" {
		t.Fatal("expected short id")
	}
	second, err := createNote(db, "second", "two")
	if err != nil {
		t.Fatal(err)
	}

	got, err := getNoteByShortID(db, first.ShortID)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Title != "first" || got.Content != "one" {
		t.Fatalf("unexpected note: %#v", got)
	}

	missing, err := getNoteByShortID(db, "missing")
	if err != nil {
		t.Fatal(err)
	}
	if missing != nil {
		t.Fatalf("expected nil missing note, got %#v", missing)
	}

	all, err := listNotes(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 || all[0].ShortID != second.ShortID || all[1].ShortID != first.ShortID {
		t.Fatalf("not newest first: %#v", all)
	}

	updated, err := updateNoteByShortID(db, first.ShortID, "updated", "changed")
	if err != nil {
		t.Fatal(err)
	}
	if updated == nil || updated.Title != "updated" || updated.Content != "changed" {
		t.Fatalf("unexpected update: %#v", updated)
	}
	updated, err = updateNoteByShortID(db, "missing", "x", "y")
	if err != nil {
		t.Fatal(err)
	}
	if updated != nil {
		t.Fatalf("expected nil updating missing, got %#v", updated)
	}

	deleted, err := deleteNoteByShortID(db, first.ShortID)
	if err != nil {
		t.Fatal(err)
	}
	if !deleted {
		t.Fatal("expected delete to report true")
	}
	deleted, err = deleteNoteByShortID(db, "missing")
	if err != nil {
		t.Fatal(err)
	}
	if deleted {
		t.Fatal("expected missing delete false")
	}
}
