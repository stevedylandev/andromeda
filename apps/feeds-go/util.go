package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func pathInt64(r *http.Request, name string) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func formatDate(ts int64) string {
	if ts <= 0 {
		return ""
	}
	return time.Unix(ts, 0).UTC().Format("Jan 2, 2006")
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	if v, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key))); err == nil {
		return v
	}
	return fallback
}

func parseIntDefault(s string, fallback int) int {
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return fallback
}

func parsePositiveInt(s string) (int, error) {
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || v < 1 {
		return 0, fmt.Errorf("invalid integer")
	}
	return v, nil
}

func validPollMinutes(v int) bool {
	return v >= 1 && v <= 1440
}

func formPollMinutes(r *http.Request) (int, bool) {
	mins, err := strconv.Atoi(r.FormValue("poll_interval_minutes"))
	return mins, err == nil && validPollMinutes(mins)
}

func itemFilterFromRequest(r *http.Request) ListItemsFilter {
	filter := ListItemsFilter{Limit: parseIntDefault(r.URL.Query().Get("limit"), 100), UnreadOnly: r.URL.Query().Get("unread") == "true"}
	if id, ok := queryInt64(r, "category_id"); ok {
		filter.CategoryID = &id
	}
	if id, ok := queryInt64(r, "subscription_id"); ok {
		filter.SubscriptionID = &id
	}
	return filter
}

func queryInt64(r *http.Request, key string) (int64, bool) {
	v := strings.TrimSpace(r.URL.Query().Get(key))
	if v == "" {
		return 0, false
	}
	id, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := []string{}
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func nullStringValue(v sql.NullString) string {
	if v.Valid {
		return v.String
	}
	return ""
}

func nullStringPointer(v sql.NullString) *string {
	if v.Valid {
		return &v.String
	}
	return nil
}

func toSubscriptionView(s Subscription) subscriptionView {
	return subscriptionView{ID: s.ID, FeedURL: s.FeedURL, Title: s.Title, SiteURL: nullStringPointer(s.SiteURL), FaviconURL: nullStringPointer(s.FaviconURL), CategoryID: func() *int64 {
		if s.CategoryID.Valid {
			return &s.CategoryID.Int64
		}
		return nil
	}(), ETag: nullStringPointer(s.ETag), LastModified: nullStringPointer(s.LastModified), LastFetchedAt: nullStringPointer(s.LastFetchedAt), LastError: nullStringPointer(s.LastError), AddedAt: s.AddedAt}
}

func loadDotEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		key := strings.TrimSpace(parts[0])
		val := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
}

func itoa(v int) string {
	return strconv.Itoa(v)
}
