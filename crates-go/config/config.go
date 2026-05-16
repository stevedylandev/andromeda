// Package config holds tiny env helpers shared by andromeda Go apps.
package config

import (
	"os"
	"strconv"
	"strings"
)

// Getenv returns the trimmed value of key or fallback when unset/blank.
func Getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

// GetenvInt parses key as int or returns fallback.
func GetenvInt(key string, fallback int) int {
	if v, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key))); err == nil {
		return v
	}
	return fallback
}

// GetenvBool returns true for "true"/"1"/"yes" (case-insensitive), false for the
// matching negatives, and fallback otherwise.
func GetenvBool(key string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	switch strings.ToLower(v) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	}
	return fallback
}

// LoadDotEnv reads a KEY=VALUE file and populates os.Environ for keys not
// already set. Missing files are silently ignored.
func LoadDotEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		key := strings.TrimSpace(parts[0])
		val := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
}
