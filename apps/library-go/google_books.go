package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type SearchHit struct {
	GoogleID string  `json:"google_id"`
	Title    string  `json:"title"`
	Authors  string  `json:"authors"`
	ISBN     *string `json:"isbn,omitempty"`
	CoverURL *string `json:"cover_url,omitempty"`
}

type volumesResponse struct {
	Items []volume `json:"items"`
}

type volume struct {
	ID         string     `json:"id"`
	VolumeInfo volumeInfo `json:"volumeInfo"`
}

type volumeInfo struct {
	Title       string       `json:"title"`
	Authors     []string     `json:"authors"`
	Identifiers []identifier `json:"industryIdentifiers"`
	ImageLinks  *imageLinks  `json:"imageLinks"`
}

type identifier struct {
	Kind       string `json:"type"`
	Identifier string `json:"identifier"`
}

type imageLinks struct {
	Thumbnail      string `json:"thumbnail"`
	SmallThumbnail string `json:"smallThumbnail"`
}

func googleBooksSearch(ctx context.Context, query, apiKey string) ([]SearchHit, error) {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return nil, nil
	}
	normalized := strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' || r == '\n' || r == '-' {
			return -1
		}
		return r
	}, trimmed)
	isISBN := (len(normalized) == 10 || len(normalized) == 13) && isISBNChars(normalized)
	q := trimmed
	if isISBN {
		q = "isbn:" + strings.ToUpper(normalized)
	}
	u := "https://www.googleapis.com/books/v1/volumes?q=" + url.QueryEscape(q) + "&maxResults=10&printType=books"
	if apiKey != "" {
		u += "&key=" + url.QueryEscape(apiKey)
	}

	client := &http.Client{Timeout: 8 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "andromeda-library-go/0.1")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("google books status %s", resp.Status)
	}
	var data volumesResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	hits := make([]SearchHit, 0, len(data.Items))
	for _, v := range data.Items {
		title := v.VolumeInfo.Title
		if title == "" {
			title = "Untitled"
		}
		hit := SearchHit{
			GoogleID: v.ID,
			Title:    title,
			Authors:  strings.Join(v.VolumeInfo.Authors, ", "),
		}
		if isbn := pickISBN(v.VolumeInfo.Identifiers); isbn != "" {
			hit.ISBN = &isbn
		}
		if v.VolumeInfo.ImageLinks != nil {
			cover := v.VolumeInfo.ImageLinks.Thumbnail
			if cover == "" {
				cover = v.VolumeInfo.ImageLinks.SmallThumbnail
			}
			if cover != "" {
				cover = strings.Replace(cover, "http://", "https://", 1)
				hit.CoverURL = &cover
			}
		}
		hits = append(hits, hit)
	}
	return hits, nil
}

func pickISBN(ids []identifier) string {
	for _, i := range ids {
		if i.Kind == "ISBN_13" {
			return i.Identifier
		}
	}
	for _, i := range ids {
		if i.Kind == "ISBN_10" {
			return i.Identifier
		}
	}
	return ""
}

func isISBNChars(s string) bool {
	for _, c := range s {
		if (c < '0' || c > '9') && c != 'X' && c != 'x' {
			return false
		}
	}
	return true
}
