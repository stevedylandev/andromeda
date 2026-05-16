package main

import (
	"html/template"
	"net/http"

	"github.com/stevedylandev/andromeda/crates-go/web"
)

func iiifURL(imageID string) string {
	return "https://www.artic.edu/iiif/2/" + imageID + "/full/843,/0/default.jpg"
}

func sourceURL(id int64) string {
	return "https://www.artic.edu/artworks/" + itoa64(id)
}

func itoa64(v int64) string {
	if v == 0 {
		return "0"
	}
	negative := v < 0
	if negative {
		v = -v
	}
	buf := [20]byte{}
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if negative {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func toArtworkView(a DailyArtwork) artworkView {
	v := artworkView{
		Date:             a.Date,
		Title:            a.Title,
		ArtistDisplay:    a.ArtistDisplay.String,
		DateDisplay:      a.DateDisplay.String,
		MediumDisplay:    a.MediumDisplay.String,
		Dimensions:       a.Dimensions.String,
		PlaceOfOrigin:    a.PlaceOfOrigin.String,
		CreditLine:       a.CreditLine.String,
		Description:      a.Description.String,
		ShortDescription: a.ShortDescription.String,
		ImageURL:         iiifURL(a.ImageID),
		SourceURL:        sourceURL(a.ArtworkID),
	}
	v.DescriptionHTML = template.HTML(v.Description)
	return v
}

func (a *App) indexHandler(w http.ResponseWriter, r *http.Request) {
	today := a.todayInTZ()
	d, err := getDaily(a.DB, today)
	if err != nil {
		a.Log.Error("index db error", "err", err)
		web.Render(a.Templates, w, "error.html", errorPageData{Title: "Error", Message: "Could not load today's artwork."}, a.Log)
		return
	}
	data := indexPageData{TodayDate: today}
	if d != nil {
		v := toArtworkView(*d)
		data.Artwork = &v
	}
	web.Render(a.Templates, w, "index.html", data, a.Log)
}

func (a *App) dayHandler(w http.ResponseWriter, r *http.Request) {
	date := r.PathValue("date")
	if _, ok := parseDate(date); !ok {
		w.WriteHeader(http.StatusBadRequest)
		web.Render(a.Templates, w, "error.html", errorPageData{Title: "Invalid date", Message: "'" + date + "' is not a valid YYYY-MM-DD date."}, a.Log)
		return
	}
	today := a.todayInTZ()
	if date > today {
		w.WriteHeader(http.StatusNotFound)
		web.Render(a.Templates, w, "error.html", errorPageData{Title: "Not yet", Message: date + " is in the future."}, a.Log)
		return
	}
	d, err := getDaily(a.DB, date)
	if err != nil {
		a.Log.Error("day db error", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		web.Render(a.Templates, w, "error.html", errorPageData{Title: "Error", Message: "Database error."}, a.Log)
		return
	}
	if d == nil {
		w.WriteHeader(http.StatusNotFound)
		web.Render(a.Templates, w, "error.html", errorPageData{Title: "Not found", Message: "No artwork stored for " + date + "."}, a.Log)
		return
	}
	web.Render(a.Templates, w, "day.html", dayPageData{Date: date, Artwork: toArtworkView(*d)}, a.Log)
}

func (a *App) archiveHandler(w http.ResponseWriter, r *http.Request) {
	items, _ := listDaily(a.DB, 1000)
	rows := make([]archiveRow, 0, len(items))
	for _, it := range items {
		artist := it.ArtistTitle.String
		if artist == "" {
			artist = it.ArtistDisplay.String
		}
		rows = append(rows, archiveRow{Date: it.Date, Title: it.Title, Artist: artist})
	}
	web.Render(a.Templates, w, "archive.html", archivePageData{Archive: rows}, a.Log)
}
