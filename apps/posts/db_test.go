package main

import (
	"database/sql"
	"testing"

	sharedsqlite "github.com/stevedylandev/andromeda/pkg/sqlite"
)

func openPostsTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sharedsqlite.Open("file:posts-test?mode=memory&cache=shared", postsSchema)
	if err != nil {
		t.Fatal(err)
	}
	seedDefaultSettings(db)
	return db
}

func strp(s string) *string { return &s }

func postInput(slug, status string) PostInput {
	return PostInput{Title: strp("Title " + slug), Slug: slug, Content: "content " + slug, Status: status, Lang: "en"}
}

func TestPostCRUDPublishedAliasesAndDisplayTitle(t *testing.T) {
	db := openPostsTestDB(t)
	defer db.Close()
	draft, err := createPost(db, postInput("draft", "draft"))
	if err != nil {
		t.Fatal(err)
	}
	pubIn := postInput("pub", "published")
	pubIn.PublishedDate = strp("2024-01-02")
	pubIn.Alias = strp("old")
	pub, err := createPost(db, pubIn)
	if err != nil {
		t.Fatal(err)
	}
	noTitle, err := createPost(db, PostInput{Slug: "no-title", Content: "body fallback content", Status: "draft", Lang: "en"})
	if err != nil {
		t.Fatal(err)
	}
	if noTitle.DisplayTitle() != "body fallback content" {
		t.Fatalf("fallback title %q", noTitle.DisplayTitle())
	}
	if _, err := createPost(db, postInput("pub", "draft")); err == nil {
		t.Fatal("expected duplicate slug failure")
	}
	got, err := getPostByShortID(db, draft.ShortID)
	if err != nil || got == nil || got.Slug != "draft" {
		t.Fatalf("get by id %#v err %v", got, err)
	}
	got, err = getPostBySlug(db, "pub")
	if err != nil || got == nil || got.ShortID != pub.ShortID {
		t.Fatalf("get by slug %#v err %v", got, err)
	}
	all, err := getAllPosts(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 || all[0].ShortID != noTitle.ShortID {
		t.Fatalf("all posts not newest first: %#v", all)
	}
	published, err := getPublishedPosts(db, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(published) != 1 || published[0].ShortID != pub.ShortID {
		t.Fatalf("published filter: %#v", published)
	}
	published, err = getPublishedPosts(db, 1)
	if err != nil || len(published) != 1 {
		t.Fatalf("published limit: %#v err %v", published, err)
	}
	redir, err := findAliasRedirect(db, "old")
	if err != nil {
		t.Fatal(err)
	}
	if redir != "/posts/pub" {
		t.Fatalf("redirect %q", redir)
	}
	redir, err = findAliasRedirect(db, "missing")
	if err != nil || redir != "" {
		t.Fatalf("missing redirect %q err %v", redir, err)
	}

	newStatus, err := togglePostStatus(db, draft.ShortID)
	if err != nil || newStatus != "published" {
		t.Fatalf("toggle draft %#v %v", newStatus, err)
	}
	newStatus, err = togglePostStatus(db, draft.ShortID)
	if err != nil || newStatus != "draft" {
		t.Fatalf("toggle published %#v %v", newStatus, err)
	}
	deleted, err := deletePost(db, pub.ShortID)
	if err != nil || !deleted {
		t.Fatalf("delete %v %v", deleted, err)
	}
	deleted, err = deletePost(db, "missing")
	if err != nil || deleted {
		t.Fatalf("delete missing %v %v", deleted, err)
	}
}

func TestPagesSettingsAndFiles(t *testing.T) {
	db := openPostsTestDB(t)
	defer db.Close()
	page, err := createPage(db, "About", "about", "hello", true, 2)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := getPageByShortID(db, page.ShortID); err != nil || got == nil || got.Title != "About" {
		t.Fatalf("page by id %#v err %v", got, err)
	}
	if got, err := getPageBySlug(db, "about"); err != nil || got == nil || got.ShortID != page.ShortID {
		t.Fatalf("page by slug %#v err %v", got, err)
	}
	updated, err := updatePage(db, page.ShortID, "About us", "about-us", "hi", false, 1)
	if err != nil || updated == nil || updated.IsPublished || updated.NavOrder != 1 {
		t.Fatalf("updated page %#v err %v", updated, err)
	}
	if err := deletePage(db, page.ShortID); err != nil {
		t.Fatal(err)
	}
	if got, err := getPageByShortID(db, page.ShortID); err != nil || got != nil {
		t.Fatalf("deleted page %#v err %v", got, err)
	}

	if v, err := getSetting(db, "missing"); err != nil || v != "" {
		t.Fatalf("missing setting %q err %v", v, err)
	}
	if err := setSetting(db, "blog_title", "Andromeda"); err != nil {
		t.Fatal(err)
	}
	if err := setSetting(db, "blog_title", "Updated"); err != nil {
		t.Fatal(err)
	}
	if v, err := getSetting(db, "blog_title"); err != nil || v != "Updated" {
		t.Fatalf("setting %q err %v", v, err)
	}

	file, err := createFile(db, "stored.jpg", "photo.jpg", "image/jpeg", 123, "")
	if err != nil {
		t.Fatal(err)
	}
	if file.StorageBackend != "local" {
		t.Fatalf("backend %q", file.StorageBackend)
	}
	if got, err := getFileByFilename(db, "stored.jpg"); err != nil || got == nil || got.OriginalName != "photo.jpg" {
		t.Fatalf("file %#v err %v", got, err)
	}
	files, err := getAllFiles(db)
	if err != nil || len(files) != 1 {
		t.Fatalf("files %#v err %v", files, err)
	}
	deleted, err := deleteFile(db, file.ShortID)
	if err != nil || deleted == nil || deleted.Filename != "stored.jpg" {
		t.Fatalf("deleted file %#v err %v", deleted, err)
	}
	deleted, err = deleteFile(db, "missing")
	if err != nil || deleted != nil {
		t.Fatalf("missing file %#v err %v", deleted, err)
	}
}
