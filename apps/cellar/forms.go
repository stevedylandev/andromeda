package main

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
)

const maxUploadBytes = 10 * 1024 * 1024

type wineFormData struct {
	Name       string
	Origin     string
	Grape      string
	Notes      string
	Background string
	Image      []byte
	ImageMime  string

	Sweetness      int
	Acidity        int
	Tannin         int
	Alcohol        int
	Body           int
	Clarity        int
	ColorIntensity int
	AromaIntensity int
	NoseComplexity int
}

func clamp1to5(v int) int {
	if v < 1 {
		return 1
	}
	if v > 5 {
		return 5
	}
	return v
}

func parseWineMultipart(r *http.Request) (*wineFormData, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		return nil, err
	}
	data := &wineFormData{
		Sweetness: 3, Acidity: 3, Tannin: 3, Alcohol: 3, Body: 3,
		Clarity: 3, ColorIntensity: 3, AromaIntensity: 3, NoseComplexity: 3,
	}
	data.Name = strings.TrimSpace(r.FormValue("name"))
	data.Origin = strings.TrimSpace(r.FormValue("origin"))
	data.Grape = strings.TrimSpace(r.FormValue("grape"))
	data.Notes = strings.TrimSpace(r.FormValue("notes"))
	data.Background = strings.TrimSpace(r.FormValue("background"))

	scoreFields := map[string]*int{
		"sweetness": &data.Sweetness, "acidity": &data.Acidity, "tannin": &data.Tannin,
		"alcohol": &data.Alcohol, "body": &data.Body,
		"clarity": &data.Clarity, "color_intensity": &data.ColorIntensity,
		"aroma_intensity": &data.AromaIntensity, "nose_complexity": &data.NoseComplexity,
	}
	for name, slot := range scoreFields {
		if v := r.FormValue(name); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				*slot = clamp1to5(n)
			}
		}
	}

	if data.Name == "" {
		return nil, errors.New("Name is required")
	}
	if err := readFormImage(r, data); err != nil {
		return nil, err
	}
	return data, nil
}

func parseWishlistMultipart(r *http.Request) (*wineFormData, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		return nil, err
	}
	data := &wineFormData{
		Sweetness: 3, Acidity: 3, Tannin: 3, Alcohol: 3, Body: 3,
		Clarity: 3, ColorIntensity: 3, AromaIntensity: 3, NoseComplexity: 3,
	}
	data.Name = strings.TrimSpace(r.FormValue("name"))
	data.Origin = strings.TrimSpace(r.FormValue("origin"))
	data.Grape = strings.TrimSpace(r.FormValue("grape"))
	data.Notes = strings.TrimSpace(r.FormValue("notes"))
	data.Background = strings.TrimSpace(r.FormValue("background"))
	if data.Name == "" {
		return nil, errors.New("Name is required")
	}
	if err := readFormImage(r, data); err != nil {
		return nil, err
	}
	return data, nil
}

func readFormImage(r *http.Request, data *wineFormData) error {
	file, _, err := r.FormFile("image")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return nil
		}
		return nil
	}
	defer file.Close()
	raw, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	if len(raw) == 0 {
		return nil
	}
	processed, err := processImage(raw)
	if err != nil {
		return err
	}
	data.Image = processed
	data.ImageMime = "image/jpeg"
	return nil
}

func formToInput(f *wineFormData) WineInput {
	return WineInput{
		Name: f.Name, Origin: f.Origin, Grape: f.Grape, Notes: f.Notes,
		Background: f.Background,
		Sweetness:  f.Sweetness, Acidity: f.Acidity, Tannin: f.Tannin,
		Alcohol: f.Alcohol, Body: f.Body,
		Clarity: f.Clarity, ColorIntensity: f.ColorIntensity,
		AromaIntensity: f.AromaIntensity, NoseComplexity: f.NoseComplexity,
	}
}

var _ multipart.File = (multipart.File)(nil)
