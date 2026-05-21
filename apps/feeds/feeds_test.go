package main

import (
	"database/sql"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sharedsqlite "github.com/stevedylandev/andromeda/pkg/sqlite"
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

func parsedEntry(guid, link string, publishedAt int64) ParsedEntry {
	return ParsedEntry{
		GUID:        guid,
		Title:       "post " + guid,
		Link:        link,
		PublishedAt: publishedAt,
	}
}

func int64Ptr(v int64) *int64 {
	return &v
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

func TestParseOPMLFlatOutlines(t *testing.T) {
	content := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0"><body>
    <outline type="rss" text="Blog A" xmlUrl="https://a.com/feed" />
    <outline type="rss" text="Blog B" xmlUrl="https://b.com/rss" />
</body></opml>`

	entries := parseOPML(content)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].XMLURL != "https://a.com/feed" || entries[0].Title != "Blog A" || entries[0].Category != "" {
		t.Fatalf("unexpected first entry: %+v", entries[0])
	}
}

func TestParseOPMLEmptyAndMissingURLs(t *testing.T) {
	if got := parseOPML(`<?xml version="1.0"?><opml><body></body></opml>`); len(got) != 0 {
		t.Fatalf("expected empty OPML to produce no entries, got %+v", got)
	}
	if got := parseOPML(`<?xml version="1.0"?><opml><body><outline type="rss" text="No URL" htmlUrl="https://example.com" /></body></opml>`); len(got) != 0 {
		t.Fatalf("expected outline without xmlUrl to be skipped, got %+v", got)
	}
}

func TestParseOPMLDeeplyNestedUsesNearestCategory(t *testing.T) {
	content := `<?xml version="1.0"?>
<opml><body>
  <outline text="Root">
    <outline text="Tech">
      <outline type="rss" text="A" xmlUrl="https://a.com/feed" />
    </outline>
    <outline type="rss" text="B" xmlUrl="https://b.com/feed" />
  </outline>
</body></opml>`

	entries := parseOPML(content)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].XMLURL != "https://a.com/feed" || entries[0].Category != "Tech" {
		t.Fatalf("unexpected deeply nested entry: %+v", entries[0])
	}
	if entries[1].XMLURL != "https://b.com/feed" || entries[1].Category != "Root" {
		t.Fatalf("unexpected root nested entry: %+v", entries[1])
	}
}

func TestParseOPMLSkipsEmptyURL(t *testing.T) {
	content := `<?xml version="1.0"?>
<opml><body>
  <outline type="rss" text="Empty" xmlUrl="" />
  <outline type="rss" text="Valid" xmlUrl="https://valid.com/feed" />
</body></opml>`

	entries := parseOPML(content)
	if len(entries) != 1 || entries[0].XMLURL != "https://valid.com/feed" {
		t.Fatalf("expected only valid URL entry, got %+v", entries)
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
	if len([]rune(got)) > 81 {
		t.Fatalf("expected title to be capped at 81 runes including ellipsis, got %d", len([]rune(got)))
	}
}

func TestDeriveTitleFromHTMLEmptyYieldsEmpty(t *testing.T) {
	for _, src := range []string{"", "<p>   </p>"} {
		if got := deriveTitleFromHTML(src); got != "" {
			t.Fatalf("expected empty derived title for %q, got %q", src, got)
		}
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

func TestCategoryCRUD(t *testing.T) {
	db := newTestDB(t)
	first, err := getOrCreateCategory(db, "  Tech  ")
	if err != nil {
		t.Fatal(err)
	}
	if first == nil || first.Name != "Tech" {
		t.Fatalf("unexpected category: %+v", first)
	}
	second, err := getOrCreateCategory(db, "Tech")
	if err != nil {
		t.Fatal(err)
	}
	if second == nil || first.ID != second.ID {
		t.Fatalf("expected same category, got first=%+v second=%+v", first, second)
	}
	other, err := getOrCreateCategory(db, "News")
	if err != nil {
		t.Fatal(err)
	}
	if other == nil || other.ID == first.ID {
		t.Fatalf("expected distinct second category, got %+v", other)
	}
	if ok, err := deleteCategory(db, first.ID); err != nil || !ok {
		t.Fatalf("delete category failed: ok=%v err=%v", ok, err)
	}
	cats, err := listCategories(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(cats) != 1 || cats[0].Name != "News" {
		t.Fatalf("unexpected categories after delete: %+v", cats)
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

func TestSettingsUpsert(t *testing.T) {
	db := newTestDB(t)
	if value, ok, err := getSetting(db, "poll"); err != nil || ok || value != "" {
		t.Fatalf("expected missing setting, value=%q ok=%v err=%v", value, ok, err)
	}
	if err := setSetting(db, "poll", "30"); err != nil {
		t.Fatal(err)
	}
	if value, ok, err := getSetting(db, "poll"); err != nil || !ok || value != "30" {
		t.Fatalf("expected setting=30, value=%q ok=%v err=%v", value, ok, err)
	}
	if err := setSetting(db, "poll", "60"); err != nil {
		t.Fatal(err)
	}
	if value, ok, err := getSetting(db, "poll"); err != nil || !ok || value != "60" {
		t.Fatalf("expected setting=60, value=%q ok=%v err=%v", value, ok, err)
	}
}

func TestSubscriptionCRUDAndMeta(t *testing.T) {
	db := newTestDB(t)
	siteURL := "https://example.com"
	sub, err := insertSubscription(db, "https://example.com/feed", "Example", &siteURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	if sub.Title != "Example" || !sub.SiteURL.Valid || sub.SiteURL.String != siteURL {
		t.Fatalf("unexpected subscription: %+v", sub)
	}

	byURL, err := getSubscriptionByURL(db, "https://example.com/feed")
	if err != nil {
		t.Fatal(err)
	}
	if byURL == nil || byURL.ID != sub.ID {
		t.Fatalf("expected subscription by URL, got %+v", byURL)
	}

	etag := "etag-1"
	lastModified := "Sun, 01 Jan 2024 00:00:00 GMT"
	if err := updateSubscriptionMeta(db, sub.ID, &etag, &lastModified, nil); err != nil {
		t.Fatal(err)
	}
	after, err := getSubscription(db, sub.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after == nil || !after.ETag.Valid || after.ETag.String != etag || !after.LastModified.Valid || after.LastModified.String != lastModified || !after.LastFetchedAt.Valid || after.LastError.Valid {
		t.Fatalf("unexpected subscription meta: %+v", after)
	}

	if ok, err := deleteSubscription(db, sub.ID); err != nil || !ok {
		t.Fatalf("delete subscription failed: ok=%v err=%v", ok, err)
	}
	gone, err := getSubscription(db, sub.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gone != nil {
		t.Fatalf("expected subscription to be deleted, got %+v", gone)
	}
}

func TestMarkReadUnread(t *testing.T) {
	db := newTestDB(t)
	sub := seedSubscriptionForTest(t, db, "https://a.com/feed", "A", nil)
	if ok, err := insertItemIgnoreDup(db, NewItem{SubscriptionID: sub.ID, GUID: "g", Title: "t", Link: "l", PublishedAt: 1}); err != nil || !ok {
		t.Fatalf("insert item failed: ok=%v err=%v", ok, err)
	}
	items, err := listItems(db, ListItemsFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %+v", items)
	}

	if ok, err := markItemRead(db, items[0].ID, true); err != nil || !ok {
		t.Fatalf("mark read failed: ok=%v err=%v", ok, err)
	}
	unread, err := listItems(db, ListItemsFilter{UnreadOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(unread) != 0 {
		t.Fatalf("expected no unread items, got %+v", unread)
	}

	if ok, err := markItemRead(db, items[0].ID, false); err != nil || !ok {
		t.Fatalf("mark unread failed: ok=%v err=%v", ok, err)
	}
	unread, err = listItems(db, ListItemsFilter{UnreadOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(unread) != 1 {
		t.Fatalf("expected one unread item, got %+v", unread)
	}
}

func TestCategoryFilterOnItems(t *testing.T) {
	db := newTestDB(t)
	tech, err := getOrCreateCategory(db, "Tech")
	if err != nil {
		t.Fatal(err)
	}
	subTech := seedSubscriptionForTest(t, db, "https://a.com/feed", "A", &tech.ID)
	subOther := seedSubscriptionForTest(t, db, "https://b.com/feed", "B", nil)
	_, _ = insertItemIgnoreDup(db, NewItem{SubscriptionID: subTech.ID, GUID: "g1", Title: "tech post", Link: "https://a.com/1", PublishedAt: 1})
	_, _ = insertItemIgnoreDup(db, NewItem{SubscriptionID: subOther.ID, GUID: "g2", Title: "other post", Link: "https://b.com/1", PublishedAt: 2})

	items, err := listItems(db, ListItemsFilter{CategoryID: &tech.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Title != "tech post" || items[0].CategoryName == nil || *items[0].CategoryName != "Tech" {
		t.Fatalf("unexpected category-filtered items: %+v", items)
	}
}

func TestSeedSubscriptionPersistsEntriesAndMeta(t *testing.T) {
	app := newTestApp(t)
	sub := seedSubscriptionForTest(t, app.DB, "https://x.com/feed", "X", nil)
	res := &FetchResult{
		ETag:         "etag-1",
		LastModified: "Sun, 01 Jan",
		Entries:      []ParsedEntry{parsedEntry("g1", "https://x.com/1", 100), parsedEntry("g2", "https://x.com/2", 200)},
	}

	if err := app.seedSubscription(sub.ID, res); err != nil {
		t.Fatal(err)
	}
	items, err := listItems(app.DB, ListItemsFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 seeded items, got %+v", items)
	}
	after, err := getSubscription(app.DB, sub.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after == nil || !after.ETag.Valid || after.ETag.String != "etag-1" || !after.LastModified.Valid || after.LastModified.String != "Sun, 01 Jan" || !after.LastFetchedAt.Valid || after.LastError.Valid {
		t.Fatalf("unexpected seeded subscription meta: %+v", after)
	}
}

func TestSeedSubscriptionSkipsEmptyLinksAndDedups(t *testing.T) {
	app := newTestApp(t)
	sub := seedSubscriptionForTest(t, app.DB, "https://x.com/feed", "X", nil)
	res := &FetchResult{Entries: []ParsedEntry{parsedEntry("g1", "", 100), parsedEntry("g2", "https://x.com/2", 200)}}
	if err := app.seedSubscription(sub.ID, res); err != nil {
		t.Fatal(err)
	}
	if err := app.seedSubscription(sub.ID, res); err != nil {
		t.Fatal(err)
	}
	items, err := listItems(app.DB, ListItemsFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].GUID != "g2" {
		t.Fatalf("expected one deduped non-empty-link item, got %+v", items)
	}
}

func TestSeedSubscriptionPrunesToItemCap(t *testing.T) {
	app := newTestApp(t)
	app.ItemCap = 3
	sub := seedSubscriptionForTest(t, app.DB, "https://x.com/feed", "X", nil)
	entries := make([]ParsedEntry, 0, 10)
	for i := int64(0); i < 10; i++ {
		entries = append(entries, parsedEntry(string(rune('a'+i)), "https://x.com/"+string(rune('a'+i)), i))
	}

	if err := app.seedSubscription(sub.ID, &FetchResult{Entries: entries}); err != nil {
		t.Fatal(err)
	}
	items, err := listItems(app.DB, ListItemsFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 || items[0].PublishedAt != 9 || items[2].PublishedAt != 7 {
		t.Fatalf("expected newest three items after prune, got %+v", items)
	}
}

func TestSeedSubscriptionWithNoEntriesStillPersistsMeta(t *testing.T) {
	app := newTestApp(t)
	sub := seedSubscriptionForTest(t, app.DB, "https://x.com/feed", "X", nil)
	if err := app.seedSubscription(sub.ID, &FetchResult{ETag: "etag-empty"}); err != nil {
		t.Fatal(err)
	}
	after, err := getSubscription(app.DB, sub.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after == nil || !after.ETag.Valid || after.ETag.String != "etag-empty" {
		t.Fatalf("expected empty seed to persist etag, got %+v", after)
	}
}

func TestResolveCategoryPrefersIDAndCreatesByName(t *testing.T) {
	app := newTestApp(t)
	existing, err := getOrCreateCategory(app.DB, "Existing")
	if err != nil {
		t.Fatal(err)
	}
	if got, err := app.resolveCategory(&existing.ID, "Ignored"); err != nil || got == nil || *got != existing.ID {
		t.Fatalf("expected existing id, got id=%v err=%v", got, err)
	}
	if got, err := app.resolveCategory(nil, "Created"); err != nil || got == nil {
		t.Fatalf("expected created category id, got id=%v err=%v", got, err)
	}
	if got, err := app.resolveCategory(nil, "   "); err != nil || got != nil {
		t.Fatalf("expected nil category for blank name, got id=%v err=%v", got, err)
	}
	if got, err := app.resolveSubscriptionCategory(updateSubscriptionBody{CategoryID: int64Ptr(existing.ID), ClearCategory: true}); err != nil || got != nil {
		t.Fatalf("expected clear category to return nil, got id=%v err=%v", got, err)
	}
}
