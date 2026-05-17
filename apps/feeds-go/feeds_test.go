package main

import (
	"database/sql"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sharedsqlite "github.com/stevedylandev/andromeda/crates-go/sqlite"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sharedsqlite.Open("file::memory:?cache=shared", feedsSchema)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func newTestApp(t *testing.T) *App {
	t.Helper()
	return &App{
		DB:                 newTestDB(t),
		Log:                slog.New(slog.NewTextHandler(io.Discard, nil)),
		DefaultPollMinutes: 30,
		ItemCap:            2,
	}
}

func seedSubscriptionForTest(t *testing.T, db *sql.DB, feedURL, title string, categoryID *int64) *Subscription {
	t.Helper()
	sub, err := insertSubscription(db, feedURL, title, nil, categoryID)
	if err != nil {
		t.Fatal(err)
	}
	return sub
}

func TestParseOPMLHandlesNestedCategories(t *testing.T) {
	content := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <body>
    <outline text="Tech">
      <outline text="Go Blog" xmlUrl="https://go.dev/feed.xml" htmlUrl="https://go.dev/blog/" />
      <outline text="News">
        <outline title="Hacker News" xmlUrl="https://hnrss.org/frontpage" htmlUrl="https://news.ycombinator.com/" />
      </outline>
    </outline>
    <outline text="Standalone" xmlUrl="https://example.com/rss.xml" />
  </body>
</opml>`

	got := parseOPML(content)
	if len(got) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(got))
	}
	if got[0].Category != "Tech" || got[0].Title != "Go Blog" {
		t.Fatalf("unexpected first entry: %+v", got[0])
	}
	if got[1].Category != "News" || got[1].Title != "Hacker News" {
		t.Fatalf("unexpected nested entry: %+v", got[1])
	}
	if got[2].Category != "" || got[2].Title != "Standalone" {
		t.Fatalf("unexpected standalone entry: %+v", got[2])
	}
}

func TestParseOPMLInvalidReturnsNil(t *testing.T) {
	if got := parseOPML("<opml><body>"); got != nil {
		t.Fatalf("expected nil for invalid OPML, got %+v", got)
	}
}

func TestDeriveTitleFromHTMLStripsMarkupAndTruncates(t *testing.T) {
	src := `<p>Hello <strong>world</strong> &amp; friends.</p>`
	if got := deriveTitleFromHTML(src); got != "Hello world & friends." {
		t.Fatalf("unexpected title: %q", got)
	}

	long := strings.Repeat("word ", 30)
	got := deriveTitleFromHTML("<div>" + long + "</div>")
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("expected ellipsis, got %q", got)
	}
}

func TestFindAlternateFeedLinksAndFavicon(t *testing.T) {
	doc := `
<html><head>
  <link rel="alternate" type="application/rss+xml" href="/rss.xml">
  <link rel="icon" type="image/png" href="/favicon.png">
  <link rel="alternate stylesheet" type="application/atom+xml" href="https://example.com/atom.xml">
</head></html>`

	links := findAlternateFeedLinks(doc)
	if len(links) != 2 {
		t.Fatalf("expected 2 feed links, got %d (%v)", len(links), links)
	}
	if links[0] != "/rss.xml" || links[1] != "https://example.com/atom.xml" {
		t.Fatalf("unexpected links: %v", links)
	}
	if href := findLinkHref(doc, func(rel, typ string) bool { return strings.Contains(strings.ToLower(rel), "icon") }); href != "/favicon.png" {
		t.Fatalf("unexpected favicon href: %q", href)
	}
}

func TestItemFilterFromRequestParsesValues(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?limit=25&unread=true&category_id=5&subscription_id=8", nil)
	filter := itemFilterFromRequest(req)
	if filter.Limit != 25 || !filter.UnreadOnly {
		t.Fatalf("unexpected base filter: %+v", filter)
	}
	if filter.CategoryID == nil || *filter.CategoryID != 5 {
		t.Fatalf("unexpected category id: %+v", filter.CategoryID)
	}
	if filter.SubscriptionID == nil || *filter.SubscriptionID != 8 {
		t.Fatalf("unexpected subscription id: %+v", filter.SubscriptionID)
	}
}

func TestFormPollMinutesValidation(t *testing.T) {
	good := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("poll_interval_minutes=60"))
	good.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if mins, ok := formPollMinutes(good); !ok || mins != 60 {
		t.Fatalf("expected valid poll minutes, got %d %v", mins, ok)
	}

	bad := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("poll_interval_minutes=0"))
	bad.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if _, ok := formPollMinutes(bad); ok {
		t.Fatal("expected invalid poll minutes")
	}
}

func TestWithCORSHandlesOptions(t *testing.T) {
	app := &App{}
	called := false
	h := app.withCORS(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	})

	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodOptions, "/api/items", nil))
	if called {
		t.Fatal("handler should not be called for OPTIONS")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("missing CORS header: %v", rec.Header())
	}
}

func TestGetOrCreateCategoryTrimsAndReuses(t *testing.T) {
	db := newTestDB(t)
	first, err := getOrCreateCategory(db, "  News  ")
	if err != nil {
		t.Fatal(err)
	}
	second, err := getOrCreateCategory(db, "News")
	if err != nil {
		t.Fatal(err)
	}
	if first == nil || second == nil || first.ID != second.ID {
		t.Fatalf("expected same category, got first=%+v second=%+v", first, second)
	}
}

func TestListItemsAndPruneSubscription(t *testing.T) {
	app := newTestApp(t)
	cat, err := getOrCreateCategory(app.DB, "Tech")
	if err != nil {
		t.Fatal(err)
	}
	sub := seedSubscriptionForTest(t, app.DB, "https://example.com/feed.xml", "Example Feed", &cat.ID)

	items := []NewItem{
		{SubscriptionID: sub.ID, GUID: "1", Title: "Old", Link: "https://example.com/1", Author: "Ron", PublishedAt: 10},
		{SubscriptionID: sub.ID, GUID: "2", Title: "Mid", Link: "https://example.com/2", Author: "", PublishedAt: 20},
		{SubscriptionID: sub.ID, GUID: "3", Title: "New", Link: "https://example.com/3", Author: "Leslie", PublishedAt: 30},
	}
	for _, item := range items {
		ok, err := insertItemIgnoreDup(app.DB, item)
		if err != nil || !ok {
			t.Fatalf("insert failed for %+v: ok=%v err=%v", item, ok, err)
		}
	}
	if ok, err := insertItemIgnoreDup(app.DB, items[0]); err != nil || ok {
		t.Fatalf("expected duplicate insert to be ignored, ok=%v err=%v", ok, err)
	}
	if _, err := markItemRead(app.DB, 1, true); err != nil {
		t.Fatal(err)
	}
	if err := pruneSubscription(app.DB, sub.ID, 2); err != nil {
		t.Fatal(err)
	}

	listed, err := listItems(app.DB, ListItemsFilter{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 2 {
		t.Fatalf("expected 2 items after prune, got %d", len(listed))
	}
	if listed[0].Title != "New" || listed[1].Title != "Mid" {
		t.Fatalf("unexpected order after prune: %+v", listed)
	}
	if listed[0].Author == nil || *listed[0].Author != "Leslie" {
		t.Fatalf("expected author pointer on newest item, got %+v", listed[0].Author)
	}
	if listed[1].Author != nil {
		t.Fatalf("expected nil author on blank author item, got %+v", listed[1].Author)
	}
	if listed[0].CategoryName == nil || *listed[0].CategoryName != "Tech" {
		t.Fatalf("expected category name, got %+v", listed[0].CategoryName)
	}

	filtered, err := listItems(app.DB, ListItemsFilter{Limit: 10, UnreadOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) != 2 {
		t.Fatalf("expected both remaining items to be unread, got %d", len(filtered))
	}
}

func TestPollIntervalMinutesUsesFallbackForMissingOrInvalidSetting(t *testing.T) {
	app := newTestApp(t)
	if got := app.pollIntervalMinutes(); got != 30 {
		t.Fatalf("expected default poll interval, got %d", got)
	}
	if err := setSetting(app.DB, "poll_interval_minutes", "45"); err != nil {
		t.Fatal(err)
	}
	if got := app.pollIntervalMinutes(); got != 45 {
		t.Fatalf("expected stored poll interval, got %d", got)
	}
	if err := setSetting(app.DB, "poll_interval_minutes", "nonsense"); err != nil {
		t.Fatal(err)
	}
	if got := app.pollIntervalMinutes(); got != 30 {
		t.Fatalf("expected fallback for invalid setting, got %d", got)
	}
}
