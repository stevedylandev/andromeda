package main

import (
	"net/http"
	"slices"
	"strings"

	"github.com/stevedylandev/andromeda/crates-go/web"
)

var commonTags = []string{"og:title", "og:description", "og:image", "og:url", "og:type"}

func (a *App) indexHandler(w http.ResponseWriter, r *http.Request) {
	web.Render(a.Templates, w, "index.html", nil, a.Log)
}

func (a *App) checkHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		web.Render(a.Templates, w, "results.html", resultsData{Error: "Bad request"}, a.Log)
		return
	}
	u := strings.TrimSpace(r.FormValue("url"))
	if u == "" {
		web.Render(a.Templates, w, "results.html", resultsData{Error: "Please enter a URL"}, a.Log)
		return
	}
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		u = "https://" + u
	}

	res, err := fetchOGData(r.Context(), u)
	if err != nil {
		web.Render(a.Templates, w, "results.html", resultsData{URL: u, Error: err.Error()}, a.Log)
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
	web.Render(a.Templates, w, "results.html", data, a.Log)
}
