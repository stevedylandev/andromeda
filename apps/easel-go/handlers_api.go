package main

import (
	"net/http"

	"github.com/stevedylandev/andromeda/crates-go/web"
)

type apiArtwork struct {
	Date             string  `json:"date"`
	ArtworkID        int64   `json:"artwork_id"`
	Title            string  `json:"title"`
	ArtistDisplay    *string `json:"artist_display,omitempty"`
	DateDisplay      *string `json:"date_display,omitempty"`
	MediumDisplay    *string `json:"medium_display,omitempty"`
	Dimensions       *string `json:"dimensions,omitempty"`
	PlaceOfOrigin    *string `json:"place_of_origin,omitempty"`
	CreditLine       *string `json:"credit_line,omitempty"`
	ShortDescription *string `json:"short_description,omitempty"`
	ImageID          string  `json:"image_id"`
	ImageURL         string  `json:"image_url"`
	SourceURL        string  `json:"source_url"`
}

func toAPI(a DailyArtwork) apiArtwork {
	out := apiArtwork{
		Date:      a.Date,
		ArtworkID: a.ArtworkID,
		Title:     a.Title,
		ImageID:   a.ImageID,
		ImageURL:  iiifURL(a.ImageID),
		SourceURL: sourceURL(a.ArtworkID),
	}
	if a.ArtistDisplay.Valid {
		v := a.ArtistDisplay.String
		out.ArtistDisplay = &v
	}
	if a.DateDisplay.Valid {
		v := a.DateDisplay.String
		out.DateDisplay = &v
	}
	if a.MediumDisplay.Valid {
		v := a.MediumDisplay.String
		out.MediumDisplay = &v
	}
	if a.Dimensions.Valid {
		v := a.Dimensions.String
		out.Dimensions = &v
	}
	if a.PlaceOfOrigin.Valid {
		v := a.PlaceOfOrigin.String
		out.PlaceOfOrigin = &v
	}
	if a.CreditLine.Valid {
		v := a.CreditLine.String
		out.CreditLine = &v
	}
	if a.ShortDescription.Valid {
		v := a.ShortDescription.String
		out.ShortDescription = &v
	}
	return out
}

func (a *App) apiToday(w http.ResponseWriter, r *http.Request) {
	d, err := getDaily(a.DB, a.todayInTZ())
	if err != nil {
		a.Log.Error("api_today db error", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if d == nil {
		web.WriteJSON(w, http.StatusNotFound, map[string]any{"error": "today not yet populated"})
		return
	}
	web.WriteJSON(w, http.StatusOK, toAPI(*d))
}

func (a *App) apiDay(w http.ResponseWriter, r *http.Request) {
	date := r.PathValue("date")
	if _, ok := parseDate(date); !ok {
		web.WriteJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid date format"})
		return
	}
	if date > a.todayInTZ() {
		web.WriteJSON(w, http.StatusNotFound, map[string]any{"error": "future date"})
		return
	}
	d, err := getDaily(a.DB, date)
	if err != nil {
		a.Log.Error("api_day db error", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if d == nil {
		web.WriteJSON(w, http.StatusNotFound, map[string]any{"error": "no record for date"})
		return
	}
	web.WriteJSON(w, http.StatusOK, toAPI(*d))
}

func (a *App) apiArchive(w http.ResponseWriter, r *http.Request) {
	items, err := listDaily(a.DB, 1000)
	if err != nil {
		a.Log.Error("api_archive db error", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	out := make([]apiArtwork, 0, len(items))
	for _, it := range items {
		out = append(out, toAPI(it))
	}
	web.WriteJSON(w, http.StatusOK, out)
}
