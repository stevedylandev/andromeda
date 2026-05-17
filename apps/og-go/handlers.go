package main

import (
	"net/http"
	"slices"
	"strings"
)

var commonTags = []string{"og:title", "og:description", "og:image", "og:url", "og:type"}

func (a *App) indexHandler(w http.ResponseWriter, r *http.Request) {
	a.renderPage(w, "index.html", nil)
}

func (a *App) checkHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.renderPage(w, "results.html", resultsData{Error: "Bad request"})
		return
	}
	u := strings.TrimSpace(r.FormValue("url"))
	if u == "" {
		a.renderPage(w, "results.html", resultsData{Error: "Please enter a URL"})
		return
	}
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		u = "https://" + u
	}

	res, err := fetchOGData(r.Context(), u)
	if err != nil {
		a.renderPage(w, "results.html", resultsData{URL: u, Error: err.Error()})
		return
	}

	data := resultsData{URL: u, OGImage: res.OGTags["og:image"], Favicon: res.Favicon}
	for _, tag := range commonTags {
		if v, ok := res.OGTags[tag]; ok {
			data.FoundTags = append(data.FoundTags, tagKV{Key: tag, Value: v})
		} else {
			data.MissingTags = append(data.MissingTags, tag)
		}
	}
	for _, key := range res.OGOrder {
		if slices.Contains(commonTags, key) {
			continue
		}
		data.FoundTags = append(data.FoundTags, tagKV{Key: key, Value: res.OGTags[key]})
	}
	data.LinkTags = res.LinkTags
	a.renderPage(w, "results.html", data)
}
