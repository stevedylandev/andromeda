package main

import (
	"net/http"
	"strconv"

	"github.com/stevedylandev/andromeda/pkg/web"
)

func (a *App) apiListQuotes(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	quotes, err := listQuotes(a.DB, limit)
	if err != nil {
		a.Log.Error("list quotes", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if quotes == nil {
		quotes = []Quote{}
	}
	web.WriteJSON(w, http.StatusOK, quotes)
}

func (a *App) apiQuoteOfTheDay(w http.ResponseWriter, r *http.Request) {
	q, err := quoteOfTheDay(a.DB)
	if err != nil {
		a.Log.Error("quote of the day", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if q == nil {
		web.WriteError(w, http.StatusNotFound, "no quotes")
		return
	}
	web.WriteJSON(w, http.StatusOK, q)
}

func (a *App) apiGetQuote(w http.ResponseWriter, r *http.Request) {
	q, err := getQuoteByShortID(a.DB, r.PathValue("short_id"))
	if err != nil {
		a.Log.Error("get quote", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if q == nil {
		web.WriteError(w, http.StatusNotFound, "not found")
		return
	}
	web.WriteJSON(w, http.StatusOK, q)
}
