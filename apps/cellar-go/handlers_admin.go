package main

import (
	"io"
	"net/http"
	"net/url"

	"github.com/stevedylandev/andromeda/crates-go/auth"
	"github.com/stevedylandev/andromeda/crates-go/web"
)

func (a *App) loginGet(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	web.Render(a.Templates, w, "login.html", loginPageData{Error: q.Get("error"), Next: q.Get("next")}, a.Log)
}

func (a *App) loginPost(w http.ResponseWriter, r *http.Request) {
	next := r.URL.Query().Get("next")
	if next == "" {
		next = "/admin"
	}
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/admin/login?error=Bad+request", http.StatusSeeOther)
		return
	}
	if !auth.VerifyPassword(r.FormValue("password"), a.AppPassword) {
		http.Redirect(w, r, "/admin/login?error=Invalid+password&next="+url.QueryEscape(next), http.StatusSeeOther)
		return
	}
	token, err := a.Sessions.Create()
	if err != nil {
		a.Log.Error("create session failed", "err", err)
		http.Redirect(w, r, "/admin/login?error=Server+error", http.StatusSeeOther)
		return
	}
	a.Sessions.PruneExpired()
	http.SetCookie(w, a.Sessions.SessionCookie(token))
	target := "/admin"
	if len(next) > 0 && next[0] == '/' {
		target = next
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(a.Sessions.CookieName); err == nil && c.Value != "" {
		a.Sessions.Delete(c.Value)
	}
	http.SetCookie(w, a.Sessions.ClearCookie())
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

func (a *App) adminIndex(w http.ResponseWriter, r *http.Request) {
	wines, err := getCellarWines(a.DB)
	if err != nil {
		a.Log.Error("list wines", "err", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	web.Render(a.Templates, w, "admin.html", adminPageData{Wines: wines}, a.Log)
}

func (a *App) newWineGet(w http.ResponseWriter, r *http.Request) {
	web.Render(a.Templates, w, "wine_form.html", wineFormPageData{Error: r.URL.Query().Get("error"), HasAnthropicKey: a.AnthropicAPIKey != ""}, a.Log)
}

func (a *App) editWineGet(w http.ResponseWriter, r *http.Request) {
	shortID := r.PathValue("short_id")
	wine, err := getWineByShortID(a.DB, shortID)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	if wine == nil {
		http.Error(w, "Wine not found", http.StatusNotFound)
		return
	}
	web.Render(a.Templates, w, "wine_form.html", wineFormPageData{Wine: wine, Error: r.URL.Query().Get("error"), HasAnthropicKey: a.AnthropicAPIKey != ""}, a.Log)
}

func (a *App) newWinePost(w http.ResponseWriter, r *http.Request) {
	data, err := parseWineMultipart(r)
	if err != nil {
		http.Redirect(w, r, "/admin/new?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	wine, err := createWine(a.DB, formToInput(data), false)
	if err != nil {
		a.Log.Error("create wine", "err", err)
		http.Redirect(w, r, "/admin/new?error=Failed+to+create+wine", http.StatusSeeOther)
		return
	}
	if len(data.Image) > 0 {
		if err := updateWineImage(a.DB, wine.ShortID, data.Image, data.ImageMime); err != nil {
			a.Log.Error("set wine image", "err", err)
		}
	}
	http.Redirect(w, r, "/wines/"+wine.ShortID, http.StatusSeeOther)
}

func (a *App) editWinePost(w http.ResponseWriter, r *http.Request) {
	shortID := r.PathValue("short_id")
	data, err := parseWineMultipart(r)
	if err != nil {
		http.Redirect(w, r, "/admin/edit/"+shortID+"?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	wine, err := updateWine(a.DB, shortID, formToInput(data))
	if err != nil {
		a.Log.Error("update wine", "err", err)
		http.Redirect(w, r, "/admin/edit/"+shortID+"?error=Failed+to+update+wine", http.StatusSeeOther)
		return
	}
	if wine == nil {
		http.Error(w, "Wine not found", http.StatusNotFound)
		return
	}
	if len(data.Image) > 0 {
		if err := updateWineImage(a.DB, shortID, data.Image, data.ImageMime); err != nil {
			a.Log.Error("update wine image", "err", err)
		}
	}
	http.Redirect(w, r, "/wines/"+shortID, http.StatusSeeOther)
}

func (a *App) deleteWinePost(w http.ResponseWriter, r *http.Request) {
	_ = deleteWine(a.DB, r.PathValue("short_id"))
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (a *App) newWishlistGet(w http.ResponseWriter, r *http.Request) {
	web.Render(a.Templates, w, "wishlist_form.html", wineFormPageData{Error: r.URL.Query().Get("error"), HasAnthropicKey: a.AnthropicAPIKey != ""}, a.Log)
}

func (a *App) editWishlistGet(w http.ResponseWriter, r *http.Request) {
	wine, err := getWineByShortID(a.DB, r.PathValue("short_id"))
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	if wine == nil {
		http.Error(w, "Wine not found", http.StatusNotFound)
		return
	}
	web.Render(a.Templates, w, "wishlist_form.html", wineFormPageData{Wine: wine, Error: r.URL.Query().Get("error"), HasAnthropicKey: a.AnthropicAPIKey != ""}, a.Log)
}

func (a *App) newWishlistPost(w http.ResponseWriter, r *http.Request) {
	data, err := parseWishlistMultipart(r)
	if err != nil {
		http.Redirect(w, r, "/admin/wishlist/new?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	wine, err := createWine(a.DB, formToInput(data), true)
	if err != nil {
		a.Log.Error("create wishlist wine", "err", err)
		http.Redirect(w, r, "/admin/wishlist/new?error=Failed+to+create+wine", http.StatusSeeOther)
		return
	}
	if len(data.Image) > 0 {
		_ = updateWineImage(a.DB, wine.ShortID, data.Image, data.ImageMime)
	}
	http.Redirect(w, r, "/wishlist", http.StatusSeeOther)
}

func (a *App) editWishlistPost(w http.ResponseWriter, r *http.Request) {
	shortID := r.PathValue("short_id")
	data, err := parseWishlistMultipart(r)
	if err != nil {
		http.Redirect(w, r, "/admin/wishlist/edit/"+shortID+"?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	wine, err := updateWishlistWine(a.DB, shortID, data.Name, data.Origin, data.Grape, data.Notes, data.Background)
	if err != nil {
		a.Log.Error("update wishlist wine", "err", err)
		http.Redirect(w, r, "/admin/wishlist/edit/"+shortID+"?error=Failed+to+update+wine", http.StatusSeeOther)
		return
	}
	if wine == nil {
		http.Error(w, "Wine not found", http.StatusNotFound)
		return
	}
	if len(data.Image) > 0 {
		_ = updateWineImage(a.DB, shortID, data.Image, data.ImageMime)
	}
	http.Redirect(w, r, "/wishlist", http.StatusSeeOther)
}

func (a *App) deleteWishlistPost(w http.ResponseWriter, r *http.Request) {
	_ = deleteWine(a.DB, r.PathValue("short_id"))
	http.Redirect(w, r, "/wishlist", http.StatusSeeOther)
}

func (a *App) promoteWinePost(w http.ResponseWriter, r *http.Request) {
	shortID := r.PathValue("short_id")
	ok, err := promoteWine(a.DB, shortID)
	if err != nil {
		http.Redirect(w, r, "/wishlist", http.StatusSeeOther)
		return
	}
	if !ok {
		http.Error(w, "Wine not found", http.StatusNotFound)
		return
	}
	http.Redirect(w, r, "/admin/edit/"+shortID, http.StatusSeeOther)
}

func (a *App) analyzeImage(w http.ResponseWriter, r *http.Request) {
	if a.AnthropicAPIKey == "" {
		web.WriteJSON(w, http.StatusBadRequest, map[string]any{"error": "No API key configured"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		web.WriteJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	file, header, err := r.FormFile("image")
	if err != nil {
		web.WriteJSON(w, http.StatusBadRequest, map[string]any{"error": "No image provided"})
		return
	}
	defer file.Close()
	raw, err := io.ReadAll(file)
	if err != nil || len(raw) == 0 {
		web.WriteJSON(w, http.StatusBadRequest, map[string]any{"error": "No image provided"})
		return
	}
	mediaType := "image/jpeg"
	if header != nil && header.Header.Get("Content-Type") != "" {
		mediaType = header.Header.Get("Content-Type")
	}
	result, err := analyzeWineImage(r.Context(), a.AnthropicAPIKey, raw, mediaType)
	if err != nil {
		a.Log.Error("Claude analysis failed", "err", err)
		web.WriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	web.WriteJSON(w, http.StatusOK, result)
}
