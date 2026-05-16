package main

import (
	"net/http"
)

func (a *App) listItemsAPI(w http.ResponseWriter, r *http.Request) {
	items, err := listItems(a.DB, itemFilterFromRequest(r))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *App) markItemReadAPI(isRead bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathInt64(r, "id")
		if !ok {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid item id"})
			return
		}
		updated, err := markItemRead(a.DB, id, isRead)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		if !updated {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "item not found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "is_read": isRead})
	}
}

func (a *App) listSubscriptionsAPI(w http.ResponseWriter, r *http.Request) {
	subs, err := listSubscriptions(a.DB)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	views := make([]subscriptionView, 0, len(subs))
	for _, sub := range subs {
		views = append(views, toSubscriptionView(sub))
	}
	writeJSON(w, http.StatusOK, map[string]any{"subscriptions": views})
}

func (a *App) createSubscriptionAPI(w http.ResponseWriter, r *http.Request) {
	var body createSubscriptionBody
	if !decodeJSONBody(w, r, &body) {
		return
	}
	sub, err := a.createSubscription(r.Context(), body, false)
	if err != nil {
		status := http.StatusBadRequest
		if isAlreadySubscribedError(err) {
			status = http.StatusConflict
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"subscription": toSubscriptionView(*sub)})
}

func (a *App) updateSubscriptionAPI(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt64(r, "id")
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid subscription id"})
		return
	}
	var body updateSubscriptionBody
	if !decodeJSONBody(w, r, &body) {
		return
	}
	categoryID, err := a.resolveSubscriptionCategory(body)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	if err := updateSubscriptionCategory(a.DB, id, categoryID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *App) deleteSubscriptionAPI(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt64(r, "id")
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid subscription id"})
		return
	}
	deleted, err := deleteSubscription(a.DB, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	if !deleted {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "subscription not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *App) listCategoriesAPI(w http.ResponseWriter, r *http.Request) {
	cats, err := listCategories(a.DB)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"categories": cats})
}

func (a *App) createCategoryAPI(w http.ResponseWriter, r *http.Request) {
	var body createCategoryBody
	if !decodeJSONBody(w, r, &body) {
		return
	}
	cat, err := getOrCreateCategory(a.DB, body.Name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"category": cat})
}

func (a *App) deleteCategoryAPI(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt64(r, "id")
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid category id"})
		return
	}
	deleted, err := deleteCategory(a.DB, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	if !deleted {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "category not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *App) importOPMLAPI(w http.ResponseWriter, r *http.Request) {
	summary, err := a.readAndImportOPML(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (a *App) getSettingsAPI(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"poll_interval_minutes": a.pollIntervalMinutes(), "default_poll_minutes": a.DefaultPollMinutes, "item_cap_per_feed": a.ItemCap, "api_key_configured": a.APIKey != ""})
}

func (a *App) updateSettingsAPI(w http.ResponseWriter, r *http.Request) {
	var body updateSettingsBody
	if !decodeJSONBody(w, r, &body) {
		return
	}
	if body.PollIntervalMinutes != nil {
		if !validPollMinutes(*body.PollIntervalMinutes) {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "poll_interval_minutes must be between 1 and 1440"})
			return
		}
		if err := setSetting(a.DB, "poll_interval_minutes", itoa(*body.PollIntervalMinutes)); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *App) discoverAPI(w http.ResponseWriter, r *http.Request) {
	var body discoverBody
	if !decodeJSONBody(w, r, &body) {
		return
	}
	feeds, err := discoverFeeds(r.Context(), body.BaseURL)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"feeds": feeds})
}
