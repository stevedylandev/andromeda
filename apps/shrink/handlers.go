package main

import (
	"io"
	"net/http"
	"strconv"

	"github.com/stevedylandev/andromeda/crates-go/web"
)

const maxUploadBytes = 20 * 1024 * 1024

func (a *App) indexHandler(w http.ResponseWriter, r *http.Request) {
	web.Render(a.Templates, w, "index.html", nil, a.Log)
}

func (a *App) compressHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		http.Error(w, "Failed to read upload: "+err.Error(), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read file: "+err.Error(), http.StatusBadRequest)
		return
	}

	quality := 80
	if v := r.FormValue("quality"); v != "" {
		if q, err := strconv.Atoi(v); err == nil {
			quality = q
		}
	}
	width := 0
	if v := r.FormValue("width"); v != "" {
		if wv, err := strconv.Atoi(v); err == nil && wv > 0 {
			width = wv
		}
	}

	out, err := compressImage(data, quality, width)
	if err != nil {
		http.Error(w, "Compression failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	name := "image"
	if header != nil && header.Filename != "" {
		name = header.Filename
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Content-Disposition", `attachment; filename="`+buildDownloadFilename(name, "jpg")+`"`)
	_, _ = w.Write(out)
}
