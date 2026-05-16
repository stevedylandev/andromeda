package main

import (
	"net/http"
	"strconv"

	"github.com/stevedylandev/andromeda/crates-go/web"
)

func (a *App) apiListBooks(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	switch status {
	case "", "all":
		status = ""
	default:
		if !validStatus(status) {
			web.WriteJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid status"})
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
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		web.WriteJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid id"})
		return
	}
	b, err := getBook(a.DB, id)
	if err != nil {
		a.Log.Error("get book", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if b == nil {
		web.WriteJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}
	web.WriteJSON(w, http.StatusOK, b)
}
