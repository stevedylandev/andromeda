package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const errAlreadySubscribed = "already subscribed"

func (a *App) createSubscription(ctx context.Context, body createSubscriptionBody, background bool) (*Subscription, error) {
	feedURL := strings.TrimSpace(body.FeedURL)
	if feedURL == "" {
		return nil, fmt.Errorf("feed_url required")
	}
	existing, err := getSubscriptionByURL(a.DB, feedURL)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf(errAlreadySubscribed)
	}
	categoryID, err := a.resolveCategory(body.CategoryID, body.CategoryName)
	if err != nil {
		return nil, err
	}
	title := firstNonEmpty(body.Title, feedURL)
	if background {
		return a.createSubscriptionInBackground(feedURL, title, body, categoryID)
	}
	return a.createSubscriptionImmediately(ctx, feedURL, title, body, categoryID)
}

func (a *App) createSubscriptionInBackground(feedURL, title string, body createSubscriptionBody, categoryID *int64) (*Subscription, error) {
	sub, err := insertSubscription(a.DB, feedURL, title, nil, categoryID)
	if err != nil {
		return nil, err
	}
	go func(subID int64, originalTitle string) {
		res, err := fetchFeed(context.Background(), feedURL, "", "")
		if err != nil {
			msg := err.Error()
			_ = updateSubscriptionMeta(a.DB, subID, nil, nil, &msg)
			return
		}
		resolvedTitle := firstNonEmpty(body.Title, res.Title, feedURL)
		if resolvedTitle != originalTitle {
			_ = updateSubscriptionTitle(a.DB, subID, resolvedTitle)
		}
		if res.SiteURL != "" {
			_ = updateSubscriptionSiteURL(a.DB, subID, &res.SiteURL)
			if fav := discoverFavicon(context.Background(), res.SiteURL); fav != "" {
				_ = updateSubscriptionFavicon(a.DB, subID, &fav)
			}
		}
		_ = a.seedSubscription(subID, res)
	}(sub.ID, sub.Title)
	return sub, nil
}

func (a *App) createSubscriptionImmediately(ctx context.Context, feedURL, title string, body createSubscriptionBody, categoryID *int64) (*Subscription, error) {
	res, err := fetchFeed(ctx, feedURL, "", "")
	if err != nil {
		return nil, fmt.Errorf("feed not reachable: %w", err)
	}
	resolvedTitle := firstNonEmpty(body.Title, res.Title, feedURL)
	siteURL := stringPtr(res.SiteURL)
	sub, err := insertSubscription(a.DB, feedURL, resolvedTitle, siteURL, categoryID)
	if err != nil {
		return nil, err
	}
	if res.SiteURL != "" {
		if fav := discoverFavicon(ctx, res.SiteURL); fav != "" {
			_ = updateSubscriptionFavicon(a.DB, sub.ID, &fav)
			sub.FaviconURL = sql.NullString{String: fav, Valid: true}
		}
	}
	if err := a.seedSubscription(sub.ID, res); err != nil {
		return nil, err
	}
	return getSubscription(a.DB, sub.ID)
}

func (a *App) seedSubscription(subID int64, res *FetchResult) error {
	for _, entry := range res.Entries {
		if strings.TrimSpace(entry.Link) == "" {
			continue
		}
		_, err := insertItemIgnoreDup(a.DB, NewItem{SubscriptionID: subID, GUID: entry.GUID, Title: entry.Title, Link: entry.Link, Author: entry.Author, PublishedAt: entry.PublishedAt})
		if err != nil {
			a.Log.Warn("seed insert failed", "sub_id", subID, "err", err)
		}
	}
	if err := pruneSubscription(a.DB, subID, a.ItemCap); err != nil {
		return err
	}
	return updateSubscriptionMeta(a.DB, subID, stringPtr(res.ETag), stringPtr(res.LastModified), nil)
}

func (a *App) resolveCategory(id *int64, name string) (*int64, error) {
	if id != nil {
		return id, nil
	}
	if strings.TrimSpace(name) == "" {
		return nil, nil
	}
	cat, err := getOrCreateCategory(a.DB, name)
	if err != nil || cat == nil {
		return nil, err
	}
	return &cat.ID, nil
}

func (a *App) resolveSubscriptionCategory(body updateSubscriptionBody) (*int64, error) {
	if body.ClearCategory {
		return nil, nil
	}
	return a.resolveCategory(body.CategoryID, body.CategoryName)
}

func (a *App) readAndImportOPML(r *http.Request) (*importSummary, error) {
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		return nil, err
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		return nil, err
	}
	defer file.Close()
	body, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	return a.importOPMLString(r.Context(), string(body)), nil
}

func (a *App) importOPMLString(ctx context.Context, content string) *importSummary {
	entries := parseOPML(content)
	summary := &importSummary{}
	for _, entry := range entries {
		existing, _ := getSubscriptionByURL(a.DB, entry.XMLURL)
		if existing != nil {
			summary.Skipped++
			continue
		}
		body := createSubscriptionBody{FeedURL: entry.XMLURL, Title: entry.Title, CategoryName: entry.Category}
		if _, err := a.createSubscription(ctx, body, true); err != nil {
			summary.Failed = append(summary.Failed, fmt.Sprintf("%s: %v", entry.XMLURL, err))
			continue
		}
		summary.Imported++
	}
	return summary
}

func (a *App) poller(ctx context.Context) {
	time.Sleep(3 * time.Second)
	for {
		mins := a.pollIntervalMinutes()
		a.Log.Info("poller pass starting", "interval_minutes", mins)
		subs, err := listSubscriptions(a.DB)
		if err == nil {
			for _, sub := range subs {
				if err := a.pollOne(ctx, sub); err != nil {
					msg := err.Error()
					_ = updateSubscriptionMeta(a.DB, sub.ID, nullStringPointer(sub.ETag), nullStringPointer(sub.LastModified), &msg)
					a.Log.Warn("feed poll failed", "feed_url", sub.FeedURL, "err", err)
				}
			}
		}
		time.Sleep(time.Duration(mins) * time.Minute)
	}
}

func (a *App) pollOne(ctx context.Context, sub Subscription) error {
	res, err := fetchFeed(ctx, sub.FeedURL, nullStringValue(sub.ETag), nullStringValue(sub.LastModified))
	if err != nil {
		return err
	}
	inserted := 0
	if res.Status != http.StatusNotModified {
		for _, entry := range res.Entries {
			ok, err := insertItemIgnoreDup(a.DB, NewItem{SubscriptionID: sub.ID, GUID: entry.GUID, Title: entry.Title, Link: entry.Link, Author: entry.Author, PublishedAt: entry.PublishedAt})
			if err != nil {
				a.Log.Warn("insert item failed", "feed_url", sub.FeedURL, "err", err)
				continue
			}
			if ok {
				inserted++
			}
		}
		if res.Title != "" && sub.Title == sub.FeedURL && res.Title != sub.Title {
			_ = updateSubscriptionTitle(a.DB, sub.ID, res.Title)
		}
		if err := pruneSubscription(a.DB, sub.ID, a.ItemCap); err != nil {
			return err
		}
	}
	if err := updateSubscriptionMeta(a.DB, sub.ID, stringPtr(res.ETag), stringPtr(res.LastModified), nil); err != nil {
		return err
	}
	a.Log.Info("feed polled", "feed_url", sub.FeedURL, "status", res.Status, "new_items", inserted, "entries", len(res.Entries))
	return nil
}

func (a *App) pollIntervalMinutes() int {
	if value, ok, err := getSetting(a.DB, "poll_interval_minutes"); err == nil && ok {
		if mins, err := parsePositiveInt(value); err == nil {
			return mins
		}
	}
	return a.DefaultPollMinutes
}

func isAlreadySubscribedError(err error) bool {
	return err != nil && strings.Contains(err.Error(), errAlreadySubscribed)
}
