package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/url"
	"time"
)

const aicSearchURL = "https://api.artic.edu/api/v1/artworks/search"

var aicFields = []string{
	"id", "title", "artist_display", "artist_title", "date_display",
	"medium_display", "dimensions", "place_of_origin", "credit_line",
	"description", "short_description", "image_id",
}

var aicExcludeFields = []string{
	"title", "description", "short_description", "term_titles", "subject_titles",
	"category_titles", "classification_titles",
}

type rawArtwork struct {
	ID               int64   `json:"id"`
	Title            *string `json:"title"`
	ArtistDisplay    *string `json:"artist_display"`
	ArtistTitle      *string `json:"artist_title"`
	DateDisplay      *string `json:"date_display"`
	MediumDisplay    *string `json:"medium_display"`
	Dimensions       *string `json:"dimensions"`
	PlaceOfOrigin    *string `json:"place_of_origin"`
	CreditLine       *string `json:"credit_line"`
	Description      *string `json:"description"`
	ShortDescription *string `json:"short_description"`
	ImageID          *string `json:"image_id"`
}

type searchResponse struct {
	Pagination struct {
		Total uint64 `json:"total"`
	} `json:"pagination"`
	Data []rawArtwork `json:"data"`
}

func buildHTTPClient() *http.Client {
	return &http.Client{Timeout: 20 * time.Second}
}

func buildAICParams(classifications, excludeTerms []string) string {
	terms := make([]string, 0, len(classifications))
	for _, c := range classifications {
		terms = append(terms, lower(c))
	}
	mustNot := make([]map[string]any, 0, len(excludeTerms))
	for _, t := range excludeTerms {
		mustNot = append(mustNot, map[string]any{
			"multi_match": map[string]any{
				"query":  t,
				"fields": aicExcludeFields,
				"type":   "phrase",
			},
		})
	}
	body := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must": []any{
					map[string]any{"term": map[string]any{"is_public_domain": true}},
					map[string]any{"terms": map[string]any{"classification_title.keyword": terms}},
					map[string]any{"exists": map[string]any{"field": "image_id"}},
				},
				"must_not": mustNot,
			},
		},
	}
	buf, _ := json.Marshal(body)
	return string(buf)
}

func lower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}

func aicTotalMatching(ctx context.Context, client *http.Client, classifications, excludeTerms []string) (uint64, error) {
	params := buildAICParams(classifications, excludeTerms)
	u := fmt.Sprintf("%s?params=%s&limit=1&fields=id", aicSearchURL, url.QueryEscape(params))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "andromeda-easel/0.1 (+https://github.com/stevedylandev/andromeda)")
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("count fetch failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("count status %s", resp.Status)
	}
	var sr searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return 0, err
	}
	return sr.Pagination.Total, nil
}

func aicFetchArtworkAt(ctx context.Context, client *http.Client, classifications, excludeTerms []string, page uint64) (*rawArtwork, error) {
	params := buildAICParams(classifications, excludeTerms)
	u := fmt.Sprintf("%s?params=%s&limit=1&page=%d&fields=%s",
		aicSearchURL, url.QueryEscape(params), page, joinFields(aicFields))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "andromeda-easel/0.1")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("artwork fetch failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("artwork status %s", resp.Status)
	}
	body := &bytes.Buffer{}
	if _, err := body.ReadFrom(resp.Body); err != nil {
		return nil, err
	}
	var sr searchResponse
	if err := json.Unmarshal(body.Bytes(), &sr); err != nil {
		return nil, err
	}
	if len(sr.Data) == 0 {
		return nil, nil
	}
	return &sr.Data[len(sr.Data)-1], nil
}

func joinFields(fs []string) string {
	out := ""
	for i, f := range fs {
		if i > 0 {
			out += ","
		}
		out += f
	}
	return out
}

func pickUnique(ctx context.Context, client *http.Client, db *sql.DB, classifications, excludeTerms []string, maxRetries int) (*rawArtwork, error) {
	total, err := aicTotalMatching(ctx, client, classifications, excludeTerms)
	if err != nil {
		return nil, err
	}
	if total == 0 {
		return nil, fmt.Errorf("AIC search returned zero matches")
	}
	for attempt := 0; attempt <= maxRetries; attempt++ {
		page := rand.Uint64N(total) + 1
		art, err := aicFetchArtworkAt(ctx, client, classifications, excludeTerms, page)
		if err != nil {
			return nil, err
		}
		if art == nil || art.ImageID == nil || *art.ImageID == "" {
			continue
		}
		exists, err := artworkIDExists(db, art.ID)
		if err != nil {
			return nil, fmt.Errorf("dedup check: %w", err)
		}
		if exists {
			continue
		}
		return art, nil
	}
	return nil, fmt.Errorf("failed to pick non-duplicate artwork after %d retries", maxRetries+1)
}

func rawToDaily(r *rawArtwork, date, fetchedAt string) *DailyArtwork {
	if r.ImageID == nil || *r.ImageID == "" {
		return nil
	}
	title := "Untitled"
	if r.Title != nil && *r.Title != "" {
		title = *r.Title
	}
	d := &DailyArtwork{
		Date:      date,
		ArtworkID: r.ID,
		Title:     title,
		ImageID:   *r.ImageID,
		FetchedAt: fetchedAt,
	}
	if r.ArtistDisplay != nil {
		d.ArtistDisplay = sql.NullString{String: *r.ArtistDisplay, Valid: true}
	}
	if r.ArtistTitle != nil {
		d.ArtistTitle = sql.NullString{String: *r.ArtistTitle, Valid: true}
	}
	if r.DateDisplay != nil {
		d.DateDisplay = sql.NullString{String: *r.DateDisplay, Valid: true}
	}
	if r.MediumDisplay != nil {
		d.MediumDisplay = sql.NullString{String: *r.MediumDisplay, Valid: true}
	}
	if r.Dimensions != nil {
		d.Dimensions = sql.NullString{String: *r.Dimensions, Valid: true}
	}
	if r.PlaceOfOrigin != nil {
		d.PlaceOfOrigin = sql.NullString{String: *r.PlaceOfOrigin, Valid: true}
	}
	if r.CreditLine != nil {
		d.CreditLine = sql.NullString{String: *r.CreditLine, Valid: true}
	}
	if r.Description != nil {
		d.Description = sql.NullString{String: *r.Description, Valid: true}
	}
	if r.ShortDescription != nil {
		d.ShortDescription = sql.NullString{String: *r.ShortDescription, Valid: true}
	}
	return d
}
