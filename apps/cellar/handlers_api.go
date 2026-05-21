package main

import (
	"net/http"

	"github.com/stevedylandev/andromeda/pkg/web"
)

func (a *App) apiListWines(w http.ResponseWriter, r *http.Request) {
	wines, err := getCellarWines(a.DB)
	if err != nil {
		a.Log.Error("api list wines", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	web.WriteJSON(w, http.StatusOK, wines)
}

func (a *App) apiGetWine(w http.ResponseWriter, r *http.Request) {
	wine, err := getWineByShortID(a.DB, r.PathValue("short_id"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if wine == nil {
		http.NotFound(w, r)
		return
	}
	web.WriteJSON(w, http.StatusOK, wine)
}

func (a *App) apiPentagonSVG(w http.ResponseWriter, r *http.Request) {
	wine, err := getWineByShortID(a.DB, r.PathValue("short_id"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if wine == nil {
		http.NotFound(w, r)
		return
	}
	svg := buildPentagonSVG(wine.Sweetness, wine.Acidity, wine.Tannin, wine.Alcohol, wine.Body, 250.0, true)
	w.Header().Set("Content-Type", "image/svg+xml")
	_, _ = w.Write([]byte(svg))
}

func (a *App) apiBarsSVG(w http.ResponseWriter, r *http.Request) {
	wine, err := getWineByShortID(a.DB, r.PathValue("short_id"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if wine == nil {
		http.NotFound(w, r)
		return
	}
	svg := buildBarsSVG(wine.Clarity, wine.ColorIntensity, wine.AromaIntensity, wine.NoseComplexity, 250.0)
	w.Header().Set("Content-Type", "image/svg+xml")
	_, _ = w.Write([]byte(svg))
}
