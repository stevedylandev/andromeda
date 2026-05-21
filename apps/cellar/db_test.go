package main

import (
	"bytes"
	"database/sql"
	"testing"

	sharedsqlite "github.com/stevedylandev/andromeda/pkg/sqlite"
)

func openCellarTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sharedsqlite.Open("file:cellar-test?mode=memory&cache=shared", cellarSchema)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func validWine(name string) WineInput {
	return WineInput{Name: name, Origin: "France", Grape: "Gamay", Notes: "notes", Background: "bg", Sweetness: 3, Acidity: 3, Tannin: 3, Alcohol: 3, Body: 3, Clarity: 3, ColorIntensity: 3, AromaIntensity: 3, NoseComplexity: 3}
}

func TestWineCRUDValidationListsAndImages(t *testing.T) {
	db := openCellarTestDB(t)
	defer db.Close()

	cellar, err := createWine(db, validWine("Cellar"), false)
	if err != nil {
		t.Fatal(err)
	}
	if cellar.ShortID == "" || cellar.Wishlist {
		t.Fatalf("unexpected cellar wine: %#v", cellar)
	}
	wish, err := createWine(db, validWine("Wish"), true)
	if err != nil {
		t.Fatal(err)
	}
	if !wish.Wishlist {
		t.Fatalf("expected wishlist wine: %#v", wish)
	}

	bad := validWine("Bad")
	bad.Sweetness = 6
	if _, err := createWine(db, bad, false); err == nil {
		t.Fatal("expected invalid sweetness to fail")
	}
	bad = validWine("Bad")
	bad.Body = 0
	if _, err := createWine(db, bad, false); err == nil {
		t.Fatal("expected zero rating to fail")
	}

	cellarList, err := getCellarWines(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(cellarList) != 1 || cellarList[0].ShortID != cellar.ShortID {
		t.Fatalf("unexpected cellar list: %#v", cellarList)
	}
	wishlist, err := getWishlistWines(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(wishlist) != 1 || wishlist[0].ShortID != wish.ShortID {
		t.Fatalf("unexpected wishlist: %#v", wishlist)
	}

	promoted, err := promoteWine(db, wish.ShortID)
	if err != nil {
		t.Fatal(err)
	}
	if !promoted {
		t.Fatal("expected promotion")
	}
	promoted, err = promoteWine(db, cellar.ShortID)
	if err != nil {
		t.Fatal(err)
	}
	if promoted {
		t.Fatal("cellar promotion should be no-op")
	}

	updatedInput := validWine("Updated")
	updated, err := updateWine(db, cellar.ShortID, updatedInput)
	if err != nil {
		t.Fatal(err)
	}
	if updated == nil || updated.Name != "Updated" {
		t.Fatalf("unexpected update: %#v", updated)
	}
	if got, err := updateWishlistWine(db, cellar.ShortID, "x", "y", "z", "n", "b"); err != nil || got != nil {
		t.Fatalf("wishlist update on cellar got %#v err %v", got, err)
	}

	img, mime, err := getWineImage(db, cellar.ShortID)
	if err != nil {
		t.Fatal(err)
	}
	if img != nil || mime != "" {
		t.Fatalf("expected no image, got %q %q", img, mime)
	}
	wantImg := []byte{1, 2, 3}
	if err := updateWineImage(db, cellar.ShortID, wantImg, "image/png"); err != nil {
		t.Fatal(err)
	}
	img, mime, err = getWineImage(db, cellar.ShortID)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(img, wantImg) || mime != "image/png" {
		t.Fatalf("bad image %v %q", img, mime)
	}

	if err := deleteWine(db, cellar.ShortID); err != nil {
		t.Fatal(err)
	}
	if got, err := getWineByShortID(db, cellar.ShortID); err != nil || got != nil {
		t.Fatalf("deleted wine got %#v err %v", got, err)
	}
	if err := deleteWine(db, "missing"); err != nil {
		t.Fatal(err)
	}
}
