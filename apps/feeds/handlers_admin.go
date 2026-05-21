package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/stevedylandev/andromeda/pkg/auth"
	"github.com/stevedylandev/andromeda/pkg/web"
)

func (a *App) loginGetHandler(w http.ResponseWriter, r *http.Request) {
	web.Render(a.Templates, w, "login.html", loginPageData{Error: r.URL.Query().Get("error")}, a.Log)
}

func (a *App) loginPostHandler(w http.ResponseWriter, r *http.Request) {
	if a.AdminPassword == "" {
		web.RedirectWithError(w, r, "/admin/login", "No admin password configured")
		return
	}
	if err := r.ParseForm(); err != nil {
		web.RedirectWithError(w, r, "/admin/login", "Bad request")
		return
	}
	if !auth.VerifyPassword(r.FormValue("password"), a.AdminPassword) {
		web.RedirectWithError(w, r, "/admin/login", "Invalid password")
		return
	}
	token, err := a.Sessions.Create()
	if err != nil {
		a.Log.Error("create session failed", "err", err)
		web.RedirectWithError(w, r, "/admin/login", "Session error")
		return
	}
	a.Sessions.PruneExpired()
	http.SetCookie(w, a.Sessions.SessionCookie(token))
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (a *App) logoutHandler(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(a.Sessions.CookieName); err == nil {
		a.Sessions.Delete(cookie.Value)
	}
	http.SetCookie(w, a.Sessions.ClearCookie())
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
	web.Render(a.Templates, w, "admin.html", adminPageData{Success: r.URL.Query().Get("success"), Error: r.URL.Query().Get("error"), Subscriptions: rows, Categories: cats, PollIntervalMinutes: a.pollIntervalMinutes(), ItemCap: a.ItemCap, APIKeyConfigured: a.APIKey != ""}, a.Log)
}

func (a *App) discoverFeedsHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		web.WriteError(w, http.StatusBadRequest, "bad request")
		return
	}
	feeds, err := discoverFeeds(r.Context(), r.FormValue("base_url"))
	if err != nil {
		web.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	web.WriteJSON(w, http.StatusOK, feeds)
}

func (a *App) addFeedHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		web.RedirectWithError(w, r, "/admin", "Bad request")
		return
	}
	body := createSubscriptionBody{FeedURL: r.FormValue("feed_url"), CategoryName: r.FormValue("category_name")}
	if _, err := a.createSubscription(r.Context(), body, true); err != nil {
		if isAlreadySubscribedError(err) {
			web.RedirectWithError(w, r, "/admin", "Already subscribed")
			return
		}
		web.RedirectWithError(w, r, "/admin", "Failed to add feed")
		return
	}
	web.RedirectWithSuccess(w, r, "/admin", "Feed added and will be fetched in the background")
}

func (a *App) deleteFeedHandler(w http.ResponseWriter, r *http.Request) {
	id, ok := web.PathInt64(r, "id")
	if !ok {
		web.RedirectWithError(w, r, "/admin", "Invalid feed ID")
		return
	}
	deleted, err := deleteSubscription(a.DB, id)
	if err != nil || !deleted {
		web.RedirectWithError(w, r, "/admin", "Failed to remove")
		return
	}
	web.RedirectWithSuccess(w, r, "/admin", "Feed removed")
}

func (a *App) updateSubCategoryHandler(w http.ResponseWriter, r *http.Request) {
	id, ok := web.PathInt64(r, "id")
	if !ok {
		web.RedirectWithError(w, r, "/admin", "Invalid feed ID")
		return
	}
	if err := r.ParseForm(); err != nil {
		web.RedirectWithError(w, r, "/admin", "Bad request")
		return
	}
	categoryID, err := a.resolveCategory(nil, r.FormValue("category_name"))
	if err != nil {
		web.RedirectWithError(w, r, "/admin", "Failed to update category")
		return
	}
	if err := updateSubscriptionCategory(a.DB, id, categoryID); err != nil {
		web.RedirectWithError(w, r, "/admin", "Failed to update category")
		return
	}
	web.RedirectWithSuccess(w, r, "/admin", "Category updated")
}

func (a *App) addCategoryHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		web.RedirectWithError(w, r, "/admin", "Bad request")
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		web.RedirectWithError(w, r, "/admin", "Name required")
		return
	}
	if _, err := getOrCreateCategory(a.DB, name); err != nil {
		web.RedirectWithError(w, r, "/admin", "Failed to add category")
		return
	}
	web.RedirectWithSuccess(w, r, "/admin", "Category added")
}

func (a *App) deleteCategoryHandler(w http.ResponseWriter, r *http.Request) {
	id, ok := web.PathInt64(r, "id")
	if !ok {
		web.RedirectWithError(w, r, "/admin", "Invalid category ID")
		return
	}
	deleted, err := deleteCategory(a.DB, id)
	if err != nil {
		web.RedirectWithError(w, r, "/admin", "Failed to remove category")
		return
	}
	if !deleted {
		web.RedirectWithError(w, r, "/admin", "Category not found")
		return
	}
	web.RedirectWithSuccess(w, r, "/admin", "Category removed")
}

func (a *App) importOPMLHandler(w http.ResponseWriter, r *http.Request) {
	summary, err := a.readAndImportOPML(r)
	if err != nil {
		web.RedirectWithError(w, r, "/admin", "No file uploaded")
		return
	}
	web.RedirectWithSuccess(w, r, "/admin", fmt.Sprintf("Imported %d, skipped %d", summary.Imported, summary.Skipped))
}

func (a *App) updateSettingsFormHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		web.RedirectWithError(w, r, "/admin", "Bad request")
		return
	}
	mins, ok := formPollMinutes(r)
	if !ok {
		web.RedirectWithError(w, r, "/admin", "Interval must be 1-1440")
		return
	}
	if err := setSetting(a.DB, "poll_interval_minutes", fmt.Sprintf("%d", mins)); err != nil {
		web.RedirectWithError(w, r, "/admin", "Failed to save settings")
		return
	}
	web.RedirectWithSuccess(w, r, "/admin", "Settings saved")
}
