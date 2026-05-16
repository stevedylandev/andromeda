package main

import (
	"net/http"
	"strings"

	"github.com/stevedylandev/andromeda/crates-go/web"
)

func (a *App) apiListCategories(w http.ResponseWriter, r *http.Request) {
	cats, err := listCategories(a.DB)
	if err != nil {
		a.Log.Error("list categories", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	web.WriteJSON(w, http.StatusOK, cats)
}

func (a *App) apiListLinks(w http.ResponseWriter, r *http.Request) {
	cats, err := listCategories(a.DB)
	if err != nil {
		a.Log.Error("list categories", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	links, err := listLinks(a.DB)
	if err != nil {
		a.Log.Error("list links", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	filter := strings.TrimSpace(r.URL.Query().Get("category"))
	if filter != "" {
		var found *Category
		for i := range cats {
			if strings.EqualFold(cats[i].Name, filter) {
				found = &cats[i]
				break
			}
		}
		if found == nil {
			web.WriteJSON(w, http.StatusNotFound, map[string]any{"error": "unknown category"})
			return
		}
		out := []Link{}
		for _, l := range links {
			if l.CategoryID == found.ID {
				out = append(out, l)
			}
		}
		web.WriteJSON(w, http.StatusOK, out)
		return
	}
	grouped := map[string][]Link{}
	for _, c := range cats {
		items := []Link{}
		for _, l := range links {
			if l.CategoryID == c.ID {
				items = append(items, l)
			}
		}
		grouped[c.Name] = items
	}
	web.WriteJSON(w, http.StatusOK, grouped)
}

func (a *App) apiCreateLink(w http.ResponseWriter, r *http.Request) {
	var body apiCreateLinkBody
	if !web.DecodeJSON(w, r, &body) {
		return
	}
	title := strings.TrimSpace(body.Title)
	url := strings.TrimSpace(body.URL)
	if title == "" || url == "" {
		web.WriteJSON(w, http.StatusBadRequest, map[string]any{"error": "title and url required"})
		return
	}
	cat, err := getCategoryByName(a.DB, body.Category)
	if err != nil {
		a.Log.Error("get category", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if cat == nil {
		web.WriteJSON(w, http.StatusNotFound, map[string]any{"error": "unknown category"})
		return
	}
	link, err := createLink(a.DB, title, url, nil, cat.ID)
	if err != nil {
		a.Log.Error("create link", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if fav := discoverFavicon(r.Context(), url); fav != "" {
		_ = updateLinkFavicon(a.DB, link.ID, &fav)
		link.FaviconURL = &fav
	}
	web.WriteJSON(w, http.StatusCreated, link)
}
