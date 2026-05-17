package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

func jpegBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 200, G: 10, B: 10, A: 255})
		}
	}
	var b bytes.Buffer
	if err := jpeg.Encode(&b, img, nil); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestBuildDownloadFilename(t *testing.T) {
	cases := map[string]string{"photo.jpg": "photo_compressed.webp", "photo": "photo_compressed.webp", "": "compressed_compressed.webp", "a.b.c.png": "a.b.c_compressed.webp"}
	for in, want := range cases {
		if got := buildDownloadFilename(in, "webp"); got != want {
			t.Fatalf("%q got %q want %q", in, got, want)
		}
	}
}

func TestCompressImage(t *testing.T) {
	if _, err := compressImage([]byte("not an image"), 80, 0); err == nil {
		t.Fatal("expected invalid image error")
	}
	out, err := compressImage(jpegBytes(t, 4, 2), 80, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 {
		t.Fatal("empty output")
	}
	resized, err := compressImage(jpegBytes(t, 4, 2), 80, 2)
	if err != nil {
		t.Fatal(err)
	}
	img, _, err := image.Decode(bytes.NewReader(resized))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 2 || img.Bounds().Dy() != 1 {
		t.Fatalf("resized bounds %v", img.Bounds())
	}
}

func makeExif(order binary.ByteOrder, magic string, gps bool) []byte {
	b := make([]byte, 40)
	copy(b[:2], magic)
	order.PutUint16(b[2:4], 42)
	order.PutUint32(b[4:8], 8)
	order.PutUint16(b[8:10], 1)
	if gps {
		order.PutUint16(b[10:12], 0x8825)
		order.PutUint32(b[18:22], 30)
		order.PutUint16(b[30:32], 7)
	} else {
		order.PutUint16(b[10:12], 0x010f)
	}
	return b
}

func TestStripGPS(t *testing.T) {
	short := []byte{1, 2, 3}
	if !bytes.Equal(stripGPS(short), short) {
		t.Fatal("short exif changed")
	}
	bad := []byte("not-tiff-data")
	if !bytes.Equal(stripGPS(bad), bad) {
		t.Fatal("bad header changed")
	}
	le := stripGPS(makeExif(binary.LittleEndian, "II", true))
	if binary.LittleEndian.Uint16(le[30:32]) != 0 {
		t.Fatal("little endian gps not zeroed")
	}
	be := stripGPS(makeExif(binary.BigEndian, "MM", true))
	if binary.BigEndian.Uint16(be[30:32]) != 0 {
		t.Fatal("big endian gps not zeroed")
	}
	noGPS := makeExif(binary.LittleEndian, "II", false)
	if !bytes.Equal(stripGPS(noGPS), noGPS) {
		t.Fatal("no gps changed")
	}
}
