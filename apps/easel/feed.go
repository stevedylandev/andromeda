package main

import (
	"net/http"
	"strings"
	"time"
)

func entryPublished(date string) string {
	if d, err := time.Parse("2006-01-02", date); err == nil {
		return time.Date(d.Year(), d.Month(), d.Day(), 12, 0, 0, 0, time.UTC).Format(time.RFC3339)
	}
	return time.Now().UTC().Format(time.RFC3339)
}

func escapeXML(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	return r.Replace(s)
}

func (a *App) atomFeedHandler(w http.ResponseWriter, r *http.Request) {
	items, err := listDaily(a.DB, 100)
	if err != nil {
		a.Log.Error("atom feed query failed", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	updated := time.Now().UTC().Format(time.RFC3339)
	if len(items) > 0 {
		updated = entryPublished(items[0].Date)
	}
	base := strings.TrimRight(a.BaseURL, "/")
	selfURL := base + "/feed.xml"

	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	b.WriteString("<feed xmlns=\"http://www.w3.org/2005/Atom\">\n")
	b.WriteString("  <title>Easel — Daily Artwork</title>\n")
	b.WriteString("  <subtitle>A daily painting from the Art Institute of Chicago</subtitle>\n")
	b.WriteString(`  <link href="` + escapeXML(selfURL) + `" rel="self" type="application/atom+xml" />` + "\n")
	b.WriteString(`  <link href="` + escapeXML(base) + `" />` + "\n")
	b.WriteString("  <id>" + escapeXML(selfURL) + "</id>\n")
	b.WriteString("  <updated>" + updated + "</updated>\n")

	for _, item := range items {
		published := entryPublished(item.Date)
		entryURL := base + "/day/" + item.Date
		author := item.ArtistTitle.String
		if author == "" {
			author = item.ArtistDisplay.String
		}
		if author == "" {
			author = "Unknown"
		}
		summary := item.ShortDescription.String
		if summary == "" {
			summary = item.Description.String
		}
		image := iiifURL(item.ImageID)
		content := `<p><img src="` + escapeXML(image) + `" alt="` + escapeXML(item.Title) + `" /></p><p>` + escapeXML(summary) + `</p>`

		b.WriteString("  <entry>\n")
		b.WriteString("    <title>" + escapeXML(item.Date) + " — " + escapeXML(item.Title) + "</title>\n")
		b.WriteString(`    <link href="` + escapeXML(entryURL) + `" />` + "\n")
		b.WriteString("    <id>" + escapeXML(entryURL) + "</id>\n")
		b.WriteString("    <updated>" + published + "</updated>\n")
		b.WriteString("    <published>" + published + "</published>\n")
		b.WriteString("    <author>\n")
		b.WriteString("      <name>" + escapeXML(author) + "</name>\n")
		b.WriteString("    </author>\n")
		if summary != "" {
			b.WriteString("    <summary>" + escapeXML(summary) + "</summary>\n")
		}
		b.WriteString(`    <content type="html">` + escapeXML(content) + `</content>` + "\n")
		b.WriteString("  </entry>\n")
	}
	b.WriteString("</feed>\n")

	w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
	_, _ = w.Write([]byte(b.String()))
}
