package main

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"
)

func compressImage(data []byte, quality int, width int) ([]byte, error) {
	exif := stripGPS(extractExif(data))
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("Failed to decode image: %w", err)
	}
	if width > 0 && width != img.Bounds().Dx() {
		src := img
		bounds := src.Bounds()
		aspect := float64(bounds.Dy()) / float64(bounds.Dx())
		height := int(float64(width)*aspect + 0.5)
		dst := image.NewRGBA(image.Rect(0, 0, width, height))
		draw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)
		img = dst
	}
	if quality < 1 {
		quality = 1
	}
	if quality > 100 {
		quality = 100
	}
	var out bytes.Buffer
	if err := jpeg.Encode(&out, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, fmt.Errorf("JPEG encoding failed: %w", err)
	}
	return injectExif(out.Bytes(), exif), nil
}

func buildDownloadFilename(original, newExt string) string {
	stem := strings.TrimSuffix(filepath.Base(original), filepath.Ext(filepath.Base(original)))
	if stem == "" {
		stem = "compressed"
	}
	return stem + "_compressed." + newExt
}
