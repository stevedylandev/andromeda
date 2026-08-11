package main

import (
	"net/http"
	"strconv"

	"github.com/stevedylandev/andromeda/pkg/web"
)

func (a *App) apiListHabits(w http.ResponseWriter, r *http.Request) {
	habits, _, err := listHabits(a.DB)
	if err != nil {
		a.Log.Error("api list habits", "err", err)
		web.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if habits == nil {
		habits = []Habit{}
	}
	web.WriteJSON(w, http.StatusOK, habits)
}

func (a *App) apiListRecords(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	records, err := listRecords(a.DB, limit)
	if err != nil {
		a.Log.Error("api list records", "err", err)
		web.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if records == nil {
		records = []Record{}
	}
	web.WriteJSON(w, http.StatusOK, records)
}
