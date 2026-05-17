package main

import (
	"html/template"
	"net/http"
	"strings"
	"time"
)

func (a *App) indexHandler(w http.ResponseWriter, r *http.Request) {
	wines, err := getCellarWines(a.DB)
	if err != nil {
		a.Log.Error("list wines", "err", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	out := make([]wineWithSVG, 0, len(wines))
	for _, wine := range wines {
		svg := buildPentagonSVG(wine.Sweetness, wine.Acidity, wine.Tannin, wine.Alcohol, wine.Body, 80.0, false)
		out = append(out, wineWithSVG{Wine: wine, PentagonSVG: template.HTML(svg)})
	}
	a.renderPage(w, "index.html", indexPageData{Wines: out})
}

func (a *App) wineDetail(w http.ResponseWriter, r *http.Request) {
	wine, err := getWineByShortID(a.DB, r.PathValue("short_id"))
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	if wine == nil {
		http.Error(w, "Wine not found", http.StatusNotFound)
		return
	}
	pentagon := buildPentagonSVG(wine.Sweetness, wine.Acidity, wine.Tannin, wine.Alcohol, wine.Body, 250.0, true)
	bars := buildBarsSVG(wine.Clarity, wine.ColorIntensity, wine.AromaIntensity, wine.NoseComplexity, 250.0)
	a.renderPage(w, "wine.html", wineDetailPageData{Wine: *wine, PentagonSVG: template.HTML(pentagon), BarsSVG: template.HTML(bars)})
}

func (a *App) wineImage(w http.ResponseWriter, r *http.Request) {
	bytes, mime, err := getWineImage(a.DB, r.PathValue("short_id"))
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	if bytes == nil {
		http.NotFound(w, r)
		return
	}
	if mime == "" {
		mime = "application/octet-stream"
	}
	w.Header().Set("Content-Type", mime)
	_, _ = w.Write(bytes)
}

func (a *App) wishlistHandler(w http.ResponseWriter, r *http.Request) {
	wines, err := getWishlistWines(a.DB)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	a.renderPage(w, "wishlist.html", wishlistPageData{Wines: wines, IsAdmin: a.Sessions.HasValid(r)})
}

func xmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;")
	return r.Replace(s)
}

func toRFC2822(sqliteTS string) string {
	if t, err := time.Parse("2006-01-02 15:04:05", sqliteTS); err == nil {
		return t.UTC().Format(time.RFC1123Z)
	}
	return sqliteTS
}

func (a *App) rssFeed(w http.ResponseWriter, r *http.Request) {
	wines, err := getCellarWines(a.DB)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	siteURL := a.SiteURL

	var items strings.Builder
	for _, wine := range wines {
		link := siteURL + "/wines/" + xmlEscape(wine.ShortID)
		title := xmlEscape(wine.Name)
		var parts []string
		if wine.Origin != "" {
			parts = append(parts, "Origin: "+wine.Origin)
		}
		if wine.Grape != "" {
			parts = append(parts, "Grape: "+wine.Grape)
		}
		if wine.Notes != "" {
			parts = append(parts, wine.Notes)
		}
		desc := xmlEscape(strings.Join(parts, " — "))
		pub := toRFC2822(wine.CreatedAt)
		items.WriteString("    <item>\n      <title>" + title + "</title>\n      <link>" + link + "</link>\n      <guid>" + link + "</guid>\n      <description>" + desc + "</description>\n      <pubDate>" + pub + "</pubDate>\n    </item>\n")
	}

	lastBuild := ""
	if len(wines) > 0 {
		lastBuild = toRFC2822(wines[0].CreatedAt)
	}

	out := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
  <channel>
    <title>` + xmlEscape(a.SiteTitle) + `</title>
    <link>` + siteURL + `</link>
    <description>` + xmlEscape(a.SiteDescription) + `</description>
    <lastBuildDate>` + lastBuild + `</lastBuildDate>
    <atom:link href="` + siteURL + `/feed.xml" rel="self" type="application/rss+xml"/>
` + items.String() + `  </channel>
</rss>`
	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	_, _ = w.Write([]byte(out))
}
