package main

import "encoding/binary"

const exifPrefix = "Exif\x00\x00"

func extractExif(orig []byte) []byte {
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
		if marker == 0xda || marker == 0xd9 { // SOS or EOI
			return nil
		}
		if marker >= 0xd0 && marker <= 0xd7 { // restart markers have no length
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
		if marker == 0xe1 && len(payload) >= len(exifPrefix) && string(payload[:len(exifPrefix)]) == exifPrefix {
			exif := make([]byte, len(payload)-len(exifPrefix))
			copy(exif, payload[len(exifPrefix):])
			return exif
		}
		pos += segLen
	}
	return nil
}

func stripGPS(exif []byte) []byte {
	if len(exif) < 8 {
		return exif
	}
	out := make([]byte, len(exif))
	copy(out, exif)

	var order binary.ByteOrder
	switch string(out[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return out
	}
	if order.Uint16(out[2:4]) != 42 {
		return out
	}
	ifd0 := int(order.Uint32(out[4:8]))
	if ifd0 < 0 || ifd0+2 > len(out) {
		return out
	}
	count := int(order.Uint16(out[ifd0 : ifd0+2]))
	entries := ifd0 + 2
	for i := 0; i < count; i++ {
		entry := entries + i*12
		if entry+12 > len(out) {
			return out
		}
		tag := order.Uint16(out[entry : entry+2])
		if tag == 0x8825 { // GPSInfo IFDPointer
			gpsOffset := int(order.Uint32(out[entry+8 : entry+12]))
			if gpsOffset >= 0 && gpsOffset+2 <= len(out) {
				order.PutUint16(out[gpsOffset:gpsOffset+2], 0)
			}
			return out
		}
	}
	return out
}

func injectExif(jpegBytes, exif []byte) []byte {
	if len(exif) == 0 || len(jpegBytes) < 2 || jpegBytes[0] != 0xff || jpegBytes[1] != 0xd8 {
		return jpegBytes
	}
	payloadLen := len(exifPrefix) + len(exif)
	segLen := payloadLen + 2
	if segLen > 0xffff {
		return jpegBytes
	}
	out := make([]byte, 0, len(jpegBytes)+segLen+2)
	out = append(out, jpegBytes[:2]...)
	out = append(out, 0xff, 0xe1, byte(segLen>>8), byte(segLen))
	out = append(out, exifPrefix...)
	out = append(out, exif...)
	out = append(out, jpegBytes[2:]...)
	return out
}
