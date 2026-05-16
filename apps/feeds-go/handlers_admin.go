package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (a *App) loginGetHandler(w http.ResponseWriter, r *http.Request) {
	a.render(w, "login.html", loginPageData{Error: r.URL.Query().Get("error")})
}

func (a *App) loginPostHandler(w http.ResponseWriter, r *http.Request) {
	if a.AdminPassword == "" {
		http.Redirect(w, r, "/admin/login?error=No+admin+password+configured", http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/admin/login?error=Bad+request", http.StatusSeeOther)
		return
	}
	if !verifyPassword(r.FormValue("password"), a.AdminPassword) {
		http.Redirect(w, r, "/admin/login?error=Invalid+password", http.StatusSeeOther)
		return
	}
	token := uuid.NewString()
	if err := createSession(a.DB, token, time.Now().Add(7*24*time.Hour)); err != nil {
		a.Log.Error("create session failed", "err", err)
		http.Redirect(w, r, "/admin/login?error=Session+error", http.StatusSeeOther)
		return
	}
	pruneExpiredSessions(a.DB)
	http.SetCookie(w, a.sessionCookie(token))
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (a *App) logoutHandler(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("feeds_session"); err == nil {
		deleteSession(a.DB, cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: "feeds_session", Value: "", Path: "/", HttpOnly: true, Secure: a.CookieSecure, SameSite: http.SameSiteLaxMode, MaxAge: -1})
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

func (a *App) adminHandler(w http.ResponseWriter, r *http.Request) {
	subs, _ := listSubscriptions(a.DB)
	cats, _ := listCategories(a.DB)
	catMap := map[int64]string{}
	for _, c := range cats {
		catMap[c.ID] = c.Name
	}
	rows := []adminSubRow{}
	for _, s := range subs {
		rows = append(rows, adminSubRow{ID: s.ID, Title: s.Title, FeedURL: s.FeedURL, SiteURL: firstNonEmpty(nullStringValue(s.SiteURL), s.FeedURL), CategoryName: catMap[s.CategoryID.Int64], LastFetchedAt: nullStringValue(s.LastFetchedAt), LastError: nullStringValue(s.LastError)})
	}
	a.render(w, "admin.html", adminPageData{Success: r.URL.Query().Get("success"), Error: r.URL.Query().Get("error"), Subscriptions: rows, Categories: cats, PollIntervalMinutes: a.pollIntervalMinutes(), ItemCap: a.ItemCap, APIKeyConfigured: a.APIKey != ""})
}

func (a *App) discoverFeedsHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "bad request"})
		return
	}
	feeds, err := discoverFeeds(r.Context(), r.FormValue("base_url"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, feeds)
}

func (a *App) addFeedHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		redirectAdminError(w, r, "Bad request")
		return
	}
	body := createSubscriptionBody{FeedURL: r.FormValue("feed_url"), CategoryName: r.FormValue("category_name")}
	if _, err := a.createSubscription(r.Context(), body, true); err != nil {
		if isAlreadySubscribedError(err) {
			redirectAdminError(w, r, "Already subscribed")
			return
		}
		redirectAdminError(w, r, "Failed to add feed")
		return
	}
	redirectAdminSuccess(w, r, "Feed added and will be fetched in the background")
}

func (a *App) deleteFeedHandler(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt64(r, "id")
	if !ok {
		redirectAdminError(w, r, "Invalid feed ID")
		return
	}
	deleted, err := deleteSubscription(a.DB, id)
	if err != nil {
		redirectAdminError(w, r, "Failed to remove")
		return
	}
	if !deleted {
		redirectAdminError(w, r, "Failed to remove")
		return
	}
	redirectAdminSuccess(w, r, "Feed removed")
}

func (a *App) updateSubCategoryHandler(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt64(r, "id")
	if !ok {
		redirectAdminError(w, r, "Invalid feed ID")
		return
	}
	if err := r.ParseForm(); err != nil {
		redirectAdminError(w, r, "Bad request")
		return
	}
	categoryID, err := a.resolveCategory(nil, r.FormValue("category_name"))
	if err != nil {
		redirectAdminError(w, r, "Failed to update category")
		return
	}
	if err := updateSubscriptionCategory(a.DB, id, categoryID); err != nil {
		redirectAdminError(w, r, "Failed to update category")
		return
	}
	redirectAdminSuccess(w, r, "Category updated")
}

func (a *App) addCategoryHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		redirectAdminError(w, r, "Bad request")
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		redirectAdminError(w, r, "Name required")
		return
	}
	if _, err := getOrCreateCategory(a.DB, name); err != nil {
		redirectAdminError(w, r, "Failed to add category")
		return
	}
	redirectAdminSuccess(w, r, "Category added")
}

func (a *App) deleteCategoryHandler(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt64(r, "id")
	if !ok {
		redirectAdminError(w, r, "Invalid category ID")
		return
	}
	deleted, err := deleteCategory(a.DB, id)
	if err != nil {
		redirectAdminError(w, r, "Failed to remove category")
		return
	}
	if !deleted {
		redirectAdminError(w, r, "Category not found")
		return
	}
	redirectAdminSuccess(w, r, "Category removed")
}

func (a *App) importOPMLHandler(w http.ResponseWriter, r *http.Request) {
	summary, err := a.readAndImportOPML(r)
	if err != nil {
		redirectAdminError(w, r, "No file uploaded")
		return
	}
	redirectAdminSuccess(w, r, fmt.Sprintf("Imported %d, skipped %d", summary.Imported, summary.Skipped))
}

func (a *App) updateSettingsFormHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		redirectAdminError(w, r, "Bad request")
		return
	}
	mins, ok := formPollMinutes(r)
	if !ok {
		redirectAdminError(w, r, "Interval must be 1-1440")
		return
	}
	if err := setSetting(a.DB, "poll_interval_minutes", fmt.Sprintf("%d", mins)); err != nil {
		redirectAdminError(w, r, "Failed to save settings")
		return
	}
	redirectAdminSuccess(w, r, "Settings saved")
}
