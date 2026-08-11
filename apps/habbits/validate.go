package main

import (
	"fmt"
	"strconv"
	"strings"
)

// normalizeValue validates raw against the habit's declared value type and
// returns the canonical stored form. The server is the source of truth; the
// UI widgets are only a convenience.
func normalizeValue(valueType, raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	switch valueType {
	case "int":
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return "", fmt.Errorf("value must be a whole number")
		}
		return strconv.FormatInt(n, 10), nil
	case "float":
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return "", fmt.Errorf("value must be a number")
		}
		return strconv.FormatFloat(f, 'f', -1, 64), nil
	case "bool":
		switch strings.ToLower(raw) {
		case "true", "1", "on", "yes", "y":
			return "true", nil
		case "false", "0", "off", "no", "n":
			return "false", nil
		default:
			return "", fmt.Errorf("value must be true or false")
		}
	case "string":
		if raw == "" {
			return "", fmt.Errorf("value is required")
		}
		return raw, nil
	default:
		return "", fmt.Errorf("unknown value type %q", valueType)
	}
}

// validValueType reports whether t is one of the allowed habit value types.
func validValueType(t string) bool {
	for _, v := range valueTypes {
		if v == t {
			return true
		}
	}
	return false
}
