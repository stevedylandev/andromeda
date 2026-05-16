package main

import (
	"net/http"
	"strconv"

	"github.com/stevedylandev/andromeda/crates-go/web"
)

const defaultListLimit int64 = 30

type apiPostSummary struct {
	ShortID         string  `json:"short_id"`
	Title           *string `json:"title"`
	Slug            string  `json:"slug"`
	PublishedDate   *string `json:"published_date"`
	MetaDescription *string `json:"meta_description"`
	MetaImage       *string `json:"meta_image"`
	CanonicalURL    *string `json:"canonical_url"`
	Lang            string  `json:"lang"`
	Tags            *string `json:"tags"`
	Content         string  `json:"content"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

type apiPostDetail struct {
	ShortID         string  `json:"short_id"`
	Title           *string `json:"title"`
	Slug            string  `json:"slug"`
	Alias           *string `json:"alias"`
	CanonicalURL    *string `json:"canonical_url"`
	PublishedDate   *string `json:"published_date"`
	MetaDescription *string `json:"meta_description"`
	MetaImage       *string `json:"meta_image"`
	Lang            string  `json:"lang"`
	Tags            *string `json:"tags"`
	Content         string  `json:"content"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

func toSummary(p Post) apiPostSummary {
	return apiPostSummary{
		ShortID: p.ShortID, Title: p.Title, Slug: p.Slug,
		PublishedDate: p.PublishedDate, MetaDescription: p.MetaDescription,
		MetaImage: p.MetaImage, CanonicalURL: p.CanonicalURL,
		Lang: p.Lang, Tags: p.Tags, Content: p.Content,
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

func toDetail(p Post) apiPostDetail {
	return apiPostDetail{
		ShortID: p.ShortID, Title: p.Title, Slug: p.Slug,
		Alias: p.Alias, CanonicalURL: p.CanonicalURL,
		PublishedDate: p.PublishedDate, MetaDescription: p.MetaDescription,
		MetaImage: p.MetaImage, Lang: p.Lang, Tags: p.Tags,
		Content: p.Content, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

func (a *App) apiListPosts(w http.ResponseWriter, r *http.Request) {
	limit := defaultListLimit
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n >= 0 {
			limit = n
		}
	}
	posts, err := getPublishedPosts(a.DB, limit)
	if err != nil {
		web.WriteJSON(w, http.StatusInternalServerError, map[string]any{"error": "internal server error"})
		return
	}
	out := make([]apiPostSummary, 0, len(posts))
	for _, p := range posts {
		out = append(out, toSummary(p))
	}
	web.WriteJSON(w, http.StatusOK, map[string]any{"posts": out})
}

func (a *App) apiGetPost(w http.ResponseWriter, r *http.Request) {
	post, err := getPostBySlug(a.DB, r.PathValue("slug"))
	if err != nil {
		web.WriteJSON(w, http.StatusInternalServerError, map[string]any{"error": "internal server error"})
		return
	}
	if post == nil || post.Status != "published" {
		web.WriteJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}
	web.WriteJSON(w, http.StatusOK, toDetail(*post))
}
