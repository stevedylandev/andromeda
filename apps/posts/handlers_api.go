package main

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/stevedylandev/andromeda/pkg/web"
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
	Weather         *string  `json:"weather"`
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
	Weather         *string  `json:"weather"`
}

func toSummary(p Post) apiPostSummary {
	return apiPostSummary{
		ShortID: p.ShortID, Title: p.Title, Slug: p.Slug,
		PublishedDate: p.PublishedDate, MetaDescription: p.MetaDescription,
		MetaImage: p.MetaImage, CanonicalURL: p.CanonicalURL,
		Lang: p.Lang, Tags: p.Tags, Content: p.Content,
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt, Weather: p.Weather,
	}
}

func toDetail(p Post) apiPostDetail {
	return apiPostDetail{
		ShortID: p.ShortID, Title: p.Title, Slug: p.Slug,
		Alias: p.Alias, CanonicalURL: p.CanonicalURL,
		PublishedDate: p.PublishedDate, MetaDescription: p.MetaDescription,
		MetaImage: p.MetaImage, Lang: p.Lang, Tags: p.Tags,
		Content: p.Content, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt, Weather: p.Weather,
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
		web.WriteError(w, http.StatusInternalServerError, "internal server error")
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
		web.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if post == nil || post.Status != "published" {
		web.WriteError(w, http.StatusNotFound, "not found")
		return
	}
	web.WriteJSON(w, http.StatusOK, toDetail(*post))
}

type apiCreatePostRequest struct {
	Title           string `json:"title"`
	Slug            string `json:"slug"`
	Content         string `json:"content"`
	Status          string `json:"status"`
	Alias           string `json:"alias"`
	CanonicalURL    string `json:"canonical_url"`
	PublishedDate   string `json:"published_date"`
	MetaDescription string `json:"meta_description"`
	MetaImage       string `json:"meta_image"`
	Lang            string `json:"lang"`
	Tags            string `json:"tags"`
	Weather         string `json:"weather"`
}

func (a *App) apiCreatePost(w http.ResponseWriter, r *http.Request) {
	var req apiCreatePostRequest
	if !web.DecodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		web.WriteError(w, http.StatusBadRequest, "content is required")
		return
	}
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "draft"
	}
	if status != "draft" && status != "published" {
		web.WriteError(w, http.StatusBadRequest, "status must be 'draft' or 'published'")
		return
	}
	title := strings.TrimSpace(req.Title)
	slug := deriveSlugWith(a, title, strings.TrimSpace(req.Slug))
	existing, err := getPostBySlug(a.DB, slug)
	if err != nil {
		web.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if existing != nil {
		web.WriteError(w, http.StatusConflict, "slug already exists")
		return
	}
	lang := "en"
	if l := strings.TrimSpace(req.Lang); l != "" {
		lang = l
	}
	weather := strings.TrimSpace(req.Weather)
	if weather == "" {
		defaultLocation, err := getSetting(a.DB, "default_location")
		if err != nil {
			defaultLocation = ""
		}
		weather = getWeather(defaultLocation)
	}
	pub := strings.TrimSpace(req.PublishedDate)
	if pub == "" {
		pub = nowDatetime()
	}
	in := PostInput{
		Title: optStr(title), Slug: slug, Content: req.Content,
		Status: status, Alias: optStr(req.Alias),
		CanonicalURL:    optStr(req.CanonicalURL),
		PublishedDate:   &pub,
		MetaDescription: optStr(req.MetaDescription),
		MetaImage:       optStr(req.MetaImage),
		Lang:            lang, Tags: optStr(req.Tags),
		Weather:         optStr(weather),
	}
	post, err := createPost(a.DB, in)
	if err != nil {
		a.Log.Error("api create post", "err", err)
		web.WriteError(w, http.StatusInternalServerError, "failed to create post")
		return
	}
	web.WriteJSON(w, http.StatusCreated, toDetail(*post))
}
