package main

import (
	"net/http"
	"strconv"

	"github.com/stevedylandev/andromeda/pkg/web"
)

func (a *App) apiListBooks(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	switch status {
	case "", "all":
		status = ""
	default:
		if !validStatus(status) {
			web.WriteError(w, http.StatusBadRequest, "invalid status")
			return
		}
	}
	books, err := listBooks(a.DB, status)
	if err != nil {
		a.Log.Error("list books", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if books == nil {
		books = []Book{}
	}
	web.WriteJSON(w, http.StatusOK, books)
}

func (a *App) apiGetBook(w http.ResponseWriter, r *http.Request) {
	id, ok := web.PathInt64(r, "id")
	if !ok {
		web.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	b, err := getBook(a.DB, id)
	if err != nil {
		a.Log.Error("get book", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if b == nil {
		web.WriteError(w, http.StatusNotFound, "not found")
		return
	}
	web.WriteJSON(w, http.StatusOK, b)
}

func (a *App) apiListQuotes(w http.ResponseWriter, r *http.Request) {
	bookID := int64(0)
	if v := r.URL.Query().Get("book_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id <= 0 {
			web.WriteError(w, http.StatusBadRequest, "invalid book_id")
			return
		}
		bookID = id
	}
	if bookID != 0 {
		quotes, err := listQuotesForBook(a.DB, bookID, true)
		if err != nil {
			a.Log.Error("list quotes for book", "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if quotes == nil {
			quotes = []Quote{}
		}
		web.WriteJSON(w, http.StatusOK, quotes)
		return
	}
	all, err := listQuotes(a.DB, true)
	if err != nil {
		a.Log.Error("list quotes", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if all == nil {
		all = []QuoteWithBook{}
	}
	web.WriteJSON(w, http.StatusOK, all)
}

func (a *App) apiGetQuote(w http.ResponseWriter, r *http.Request) {
	id, ok := web.PathInt64(r, "id")
	if !ok {
		web.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	q, err := getQuote(a.DB, id)
	if err != nil {
		a.Log.Error("get quote", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if q == nil || !q.Public {
		web.WriteError(w, http.StatusNotFound, "not found")
		return
	}
	web.WriteJSON(w, http.StatusOK, q)
}
