package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	_ "image/png"
)

func processImage(data []byte) ([]byte, error) {
	orientation := readOrientation(data)
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("Failed to decode image: %w", err)
	}
	img = applyOrientation(img, orientation)
	var out bytes.Buffer
	if err := jpeg.Encode(&out, img, &jpeg.Options{Quality: 75}); err != nil {
		return nil, fmt.Errorf("JPEG encoding failed: %w", err)
	}
	return out.Bytes(), nil
}

func readOrientation(data []byte) int {
	exif := extractExifSegment(data)
	if len(exif) < 8 {
		return 1
	}
	var order binary.ByteOrder
	switch string(exif[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 1
	}
	if order.Uint16(exif[2:4]) != 42 {
		return 1
	}
	ifd0 := int(order.Uint32(exif[4:8]))
	if ifd0 < 0 || ifd0+2 > len(exif) {
		return 1
	}
	count := int(order.Uint16(exif[ifd0 : ifd0+2]))
	entries := ifd0 + 2
	for i := 0; i < count; i++ {
		entry := entries + i*12
		if entry+12 > len(exif) {
			return 1
		}
		tag := order.Uint16(exif[entry : entry+2])
		if tag == 0x0112 { // Orientation
			return int(order.Uint16(exif[entry+8 : entry+10]))
		}
	}
	return 1
}

func extractExifSegment(orig []byte) []byte {
	const prefix = "Exif\x00\x00"
	if len(orig) < 4 || orig[0] != 0xff || orig[1] != 0xd8 {
		return nil
	}
	pos := 2
	for pos+4 <= len(orig) {
		if orig[pos] != 0xff {
			return nil
		}
		marker := orig[pos+1]
		pos += 2
		if marker == 0xda || marker == 0xd9 {
			return nil
		}
		if marker >= 0xd0 && marker <= 0xd7 {
			continue
		}
		if pos+2 > len(orig) {
			return nil
		}
		segLen := int(binary.BigEndian.Uint16(orig[pos : pos+2]))
		if segLen < 2 || pos+segLen > len(orig) {
			return nil
		}
		payload := orig[pos+2 : pos+segLen]
		if marker == 0xe1 && len(payload) >= len(prefix) && string(payload[:len(prefix)]) == prefix {
			return payload[len(prefix):]
		}
		pos += segLen
	}
	return nil
}

func applyOrientation(src image.Image, orientation int) image.Image {
	if orientation <= 1 || orientation > 8 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(rgba, rgba.Bounds(), src, b.Min, draw.Src)

	var dst *image.RGBA
	switch orientation {
	case 2: // flip horizontal
		dst = image.NewRGBA(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dst.Set(w-1-x, y, rgba.At(x, y))
			}
		}
	case 3: // rotate 180
		dst = image.NewRGBA(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dst.Set(w-1-x, h-1-y, rgba.At(x, y))
			}
		}
	case 4: // flip vertical
		dst = image.NewRGBA(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dst.Set(x, h-1-y, rgba.At(x, y))
			}
		}
	case 5: // transpose
		dst = image.NewRGBA(image.Rect(0, 0, h, w))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dst.Set(y, x, rgba.At(x, y))
			}
		}
	case 6: // rotate 90 CW
		dst = image.NewRGBA(image.Rect(0, 0, h, w))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dst.Set(h-1-y, x, rgba.At(x, y))
			}
		}
	case 7: // transverse
		dst = image.NewRGBA(image.Rect(0, 0, h, w))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dst.Set(h-1-y, w-1-x, rgba.At(x, y))
			}
		}
	case 8: // rotate 270 CW (90 CCW)
		dst = image.NewRGBA(image.Rect(0, 0, h, w))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dst.Set(y, w-1-x, rgba.At(x, y))
			}
		}
	}
	return dst
}
