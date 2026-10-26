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
    OLKey    string  `json:"key"`
    Title    string  `json:"title"`
    Authors  string  `json:"author_name"`
    ISBN     *string `json:"isbn,omitempty"` // canonical ISBN-13
    CoverURL *string `json:"cover_url,omitempty"`
}

type olResponse struct {
    Docs []olDoc `json:"docs"`
}

type olDoc struct {
    OLKey   string   `json:"key"`
    Title   string   `json:"title"`
    Authors []string `json:"author_name"`
    ISBN    []string `json:"isbn"`
    CoverID int64    `json:"cover_i"`
}

func openLibrarySearch(ctx context.Context, query string) ([]SearchHit, error) {
    trimmed := strings.TrimSpace(query)
    if trimmed == "" {
        return nil, nil
    }
    q := trimmed
    if isISBNQuery(trimmed) {
        // Open Library normalizes both ISBN-10 and ISBN-13 here; just uppercase.
        q = "isbn:" + strings.ToUpper(stripISBNSep(trimmed))
    }
    u := "https://openlibrary.org/search.json?" + url.Values{
        "q":      {q},
        "limit":  {"10"},
        // Explicit fields: since Jan 2025 the default field set is reduced,
        // so you must ask for what you use.
        "fields": {"key,title,author_name,isbn,cover_i"},
    }.Encode()

    client := &http.Client{Timeout: 8 * time.Second}
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
    if err != nil {
        return nil, err
    }
    req.Header.Set("User-Agent", "andromeda-library/0.1") // no API key needed
    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("request: %w", err)
    }
    defer resp.Body.Close()
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return nil, fmt.Errorf("openlibrary status %s", resp.Status)
    }
    var data olResponse
    if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
        return nil, fmt.Errorf("parse: %w", err)
    }

    hits := make([]SearchHit, 0, len(data.Docs))
    for _, d := range data.Docs {
        title := d.Title
        if title == "" {
            title = "Untitled"
        }
        hit := SearchHit{
            OLKey:   d.OLKey,
            Title:   title,
            Authors: strings.Join(d.Authors, ", "),
        }
        if isbn := pickISBN(d.ISBN); isbn != "" {
            hit.ISBN = &isbn
        }
        if d.CoverID != 0 {
            cover := fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-L.jpg", d.CoverID)
            hit.CoverURL = &cover
        }
        hits = append(hits, hit)
    }
    return hits, nil
}


func isISBNChars(s string) bool {
	for _, c := range s {
		if (c < '0' || c > '9') && c != 'X' && c != 'x' {
			return false
		}
	}
	return true
}

func isISBNQuery(s string) bool {
    n := stripISBNSep(s)
    return (len(n) == 10 || len(n) == 13) && isISBNChars(n)
}

func stripISBNSep(s string) string {
    return strings.Map(func(r rune) rune {
        if r == '-' || r == ' ' {
            return -1
        }
        return r
    }, s)
}

// pickISBN normalizes to ISBN-13 and returns one canonical value.
func pickISBN(raw []string) string {
    seen := map[string]struct{}{}
    var firstValid string
    for _, r := range raw {
        n := strings.ToUpper(stripISBNSep(r))
        if len(n) == 10 {
            if !validISBN10(n) {
                continue
            }
            n, _ = isbn10to13(n)
        }
        if len(n) != 13 || !validISBN13(n) {
            continue
        }
        if _, dup := seen[n]; dup {
            continue
        }
        seen[n] = struct{}{}
        if firstValid == "" {
            firstValid = n
        }
    }
    return firstValid
}

func validISBN13(s string) bool {
    sum := 0
    for i, c := range s {
        d := int(c - '0')
        if d < 0 || d > 9 {
            return false
        }
        if i%2 == 1 {
            d *= 3
        }
        sum += d
    }
    return sum%10 == 0
}

func validISBN10(s string) bool {
    sum := 0
    for i, c := range s {
        d := 10
        if c != 'X' {
            d = int(c - '0')
            if d < 0 || d > 9 {
                return false
            }
        } else if i != 9 {
            return false // X only valid as check digit
        }
        sum += d * (10 - i)
    }
    return sum%11 == 0
}

func isbn10to13(s string) (string, bool) {
    if len(s) != 10 {
        return "", false
    }
    base := "978" + s[:9]
    sum := 0
    for i, c := range base {
        d := int(c - '0')
        if d < 0 || d > 9 {
            return "", false
        }
        if i%2 == 1 {
            d *= 3
        }
        sum += d
    }
    return base + string(rune('0'+(10-sum%10)%10)), true
}

