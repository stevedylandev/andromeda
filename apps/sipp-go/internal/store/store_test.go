package store

import "testing"

func TestSnippetCRUDAndOrdering(t *testing.T) {
	db, err := Open("file:sipp-store-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	first, err := Create(db, "first", "one")
	if err != nil {
		t.Fatal(err)
	}
	if first.ShortID == "" {
		t.Fatal("expected short id")
	}
	second, err := Create(db, "second", "two")
	if err != nil {
		t.Fatal(err)
	}

	got, err := GetByShortID(db, first.ShortID)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Name != "first" || got.Content != "one" {
		t.Fatalf("unexpected snippet: %#v", got)
	}
	missing, err := GetByShortID(db, "missing")
	if err != nil {
		t.Fatal(err)
	}
	if missing != nil {
		t.Fatalf("expected nil missing snippet, got %#v", missing)
	}

	all, err := List(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 || all[0].ShortID != second.ShortID || all[1].ShortID != first.ShortID {
		t.Fatalf("not newest first: %#v", all)
	}

	updated, err := UpdateByShortID(db, first.ShortID, "updated", "changed")
	if err != nil {
		t.Fatal(err)
	}
	if updated == nil || updated.Name != "updated" || updated.Content != "changed" {
		t.Fatalf("unexpected update: %#v", updated)
	}
	updated, err = UpdateByShortID(db, "missing", "x", "y")
	if err != nil {
		t.Fatal(err)
	}
	if updated != nil {
		t.Fatalf("expected nil updating missing, got %#v", updated)
	}

	deleted, err := DeleteByShortID(db, first.ShortID)
	if err != nil {
		t.Fatal(err)
	}
	if !deleted {
		t.Fatal("expected delete true")
	}
	deleted, err = DeleteByShortID(db, "missing")
	if err != nil {
		t.Fatal(err)
	}
	if deleted {
		t.Fatal("expected missing delete false")
	}
}
