package main

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/stevedylandev/andromeda/pkg/web"
)

const opmlPreviewPerFeed = 5

func (a *App) indexHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("url")
	if query == "" {
		query = r.URL.Query().Get("urls")
	}
	data := indexPageData{BaseURL: a.BaseURL}
	if query != "" {
		urls := splitAndTrim(query)
		data.FeedURLs = urls
		if len(urls) == 0 {
			data.Error = "No URLs provided"
			web.Render(a.Templates, w, "index.html", data, a.Log)
			return
		}
		perFeed := 0
		if len(urls) == 1 && isOPMLURL(urls[0]) {
			content, err := fetchOPMLDoc(r.Context(), urls[0])
			if err != nil {
				a.Log.Warn("opml preview fetch failed", "url", urls[0], "err", err)
				data.Error = "Could not load OPML"
				web.Render(a.Templates, w, "index.html", data, a.Log)
				return
			}
			entries := parseOPML(content)
			if len(entries) == 0 {
				data.Error = "No feeds found in OPML"
				web.Render(a.Templates, w, "index.html", data, a.Log)
				return
			}
			feedURLs := make([]string, 0, len(entries))
			for _, e := range entries {
				if e.XMLURL != "" {
					feedURLs = append(feedURLs, e.XMLURL)
				}
			}
			urls = feedURLs
			perFeed = opmlPreviewPerFeed
		}
		for _, item := range previewURLs(r.Context(), urls, perFeed, a.Log) {
			data.Items = append(data.Items, templateItem{Title: item.Title, Link: item.Link, Author: item.Author, FormattedDate: formatDate(item.Published)})
		}
		web.Render(a.Templates, w, "index.html", data, a.Log)
		return
	}

	items, err := listItems(a.DB, ListItemsFilter{Limit: 100})
	if err != nil {
		a.Log.Error("index query failed", "err", err)
		data.Error = "Error loading feeds. Please try again later."
		web.Render(a.Templates, w, "index.html", data, a.Log)
		return
	}
	for _, item := range items {
		author := item.FeedTitle
		if item.Author != nil && strings.TrimSpace(*item.Author) != "" {
			author = item.FeedTitle + " - " + *item.Author
		}
		data.Items = append(data.Items, templateItem{Title: item.Title, Link: item.Link, Author: author, FormattedDate: formatDate(item.PublishedAt)})
	}
	web.Render(a.Templates, w, "index.html", data, a.Log)
}

func (a *App) feedsExportHandler(w http.ResponseWriter, r *http.Request) {
	subs, err := listSubscriptions(a.DB)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	switch r.URL.Query().Get("format") {
	case "", "json":
		rows := make([]map[string]any, 0, len(subs))
		for _, s := range subs {
			rows = append(rows, map[string]any{
				"id":      fmt.Sprintf("feed/%d", s.ID),
				"title":   s.Title,
				"url":     s.FeedURL,
				"htmlUrl": nullStringValue(s.SiteURL),
			})
		}
		web.WriteJSON(w, http.StatusOK, map[string]any{"subscriptions": rows})
	case "opml":
		a.writeOPMLExport(w, subs)
	default:
		web.WriteError(w, http.StatusBadRequest, "Invalid format. Use ?format=json or ?format=opml")
	}
}

func (a *App) atomFeedHandler(w http.ResponseWriter, r *http.Request) {
	items, err := listItems(a.DB, ListItemsFilter{Limit: 100})
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	type atomLink struct {
		Href string `xml:"href,attr"`
		Rel  string `xml:"rel,attr,omitempty"`
		Type string `xml:"type,attr,omitempty"`
	}
	type atomAuthor struct {
		Name string `xml:"name"`
	}
	type atomSource struct {
		Title string `xml:"title"`
	}
	type atomEntry struct {
		Title     string     `xml:"title"`
		Link      atomLink   `xml:"link"`
		ID        string     `xml:"id"`
		Updated   string     `xml:"updated"`
		Published string     `xml:"published"`
		Author    atomAuthor `xml:"author"`
		Source    atomSource `xml:"source"`
	}
	type atomFeed struct {
		XMLName xml.Name    `xml:"feed"`
		Xmlns   string      `xml:"xmlns,attr"`
		Title   string      `xml:"title"`
		Links   []atomLink  `xml:"link"`
		ID      string      `xml:"id"`
		Updated string      `xml:"updated"`
		Entries []atomEntry `xml:"entry"`
	}

	updated := time.Now().UTC().Format(time.RFC3339)
	if len(items) > 0 {
		updated = time.Unix(items[0].PublishedAt, 0).UTC().Format(time.RFC3339)
	}
	base := strings.TrimRight(a.BaseURL, "/")
	feed := atomFeed{
		Xmlns:   "http://www.w3.org/2005/Atom",
		Title:   "Feeds",
		ID:      base + "/feed.xml",
		Updated: updated,
		Links:   []atomLink{{Href: base + "/feed.xml", Rel: "self", Type: "application/atom+xml"}, {Href: base}},
	}
	for _, item := range items {
		author := item.FeedTitle
		if item.Author != nil && *item.Author != "" {
			author = *item.Author
		}
		entryID := item.GUID
		if strings.TrimSpace(entryID) == "" {
			entryID = item.Link
		}
		published := time.Unix(item.PublishedAt, 0).UTC().Format(time.RFC3339)
		feed.Entries = append(feed.Entries, atomEntry{Title: item.Title, Link: atomLink{Href: item.Link}, ID: entryID, Updated: published, Published: published, Author: atomAuthor{Name: author}, Source: atomSource{Title: item.FeedTitle}})
	}
	body, _ := xml.MarshalIndent(feed, "", "  ")
	w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
	_, _ = w.Write([]byte(xml.Header))
	_, _ = w.Write(body)
}

func (a *App) writeOPMLExport(w http.ResponseWriter, subs []Subscription) {
	cats, _ := listCategories(a.DB)
	catNames := map[int64]string{}
	for _, c := range cats {
		catNames[c.ID] = c.Name
	}
	grouped := map[string][]Subscription{}
	for _, s := range subs {
		key := ""
		if s.CategoryID.Valid {
			key = catNames[s.CategoryID.Int64]
		}
		grouped[key] = append(grouped[key], s)
	}
	type outline struct {
		XMLName xml.Name  `xml:"outline"`
		Text    string    `xml:"text,attr,omitempty"`
		Title   string    `xml:"title,attr,omitempty"`
		Type    string    `xml:"type,attr,omitempty"`
		XMLURL  string    `xml:"xmlUrl,attr,omitempty"`
		HTMLURL string    `xml:"htmlUrl,attr,omitempty"`
		Nodes   []outline `xml:"outline,omitempty"`
	}
	type opml struct {
		XMLName xml.Name `xml:"opml"`
		Version string   `xml:"version,attr"`
		Head    struct {
			Title       string `xml:"title"`
			DateCreated string `xml:"dateCreated"`
		} `xml:"head"`
		Body struct {
			Nodes []outline `xml:"outline"`
		} `xml:"body"`
	}
	doc := opml{Version: "2.0"}
	doc.Head.Title = "Feeds"
	doc.Head.DateCreated = time.Now().Format(time.RFC1123Z)
	for category, rows := range grouped {
		if category == "" {
			for _, s := range rows {
				doc.Body.Nodes = append(doc.Body.Nodes, outline{Text: s.Title, Title: s.Title, Type: "rss", XMLURL: s.FeedURL, HTMLURL: nullStringValue(s.SiteURL)})
			}
			continue
		}
		group := outline{Text: category, Title: category}
		for _, s := range rows {
			group.Nodes = append(group.Nodes, outline{Text: s.Title, Title: s.Title, Type: "rss", XMLURL: s.FeedURL, HTMLURL: nullStringValue(s.SiteURL)})
		}
		doc.Body.Nodes = append(doc.Body.Nodes, group)
	}
	body, _ := xml.MarshalIndent(doc, "", "  ")
	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("Content-Disposition", `attachment; filename="feeds.opml"`)
	_, _ = w.Write([]byte(xml.Header))
	_, _ = w.Write(body)
}

func isOPMLURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return strings.HasSuffix(strings.ToLower(u.Path), ".opml")
}
