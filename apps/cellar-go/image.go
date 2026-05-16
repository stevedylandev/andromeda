package main

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
)

func processImage(data []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("Failed to decode image: %w", err)
	}
	var out bytes.Buffer
	if err := jpeg.Encode(&out, img, &jpeg.Options{Quality: 75}); err != nil {
		return nil, fmt.Errorf("JPEG encoding failed: %w", err)
	}
	return out.Bytes(), nil
}
