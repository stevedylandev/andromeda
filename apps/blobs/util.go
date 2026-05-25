package main

import (
	"net/url"
	"path"
	"strings"
)

func isImageName(name string) bool {
	switch strings.ToLower(path.Ext(name)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".avif", ".bmp", ".ico":
		return true
	}
	return false
}

func humanSize(n int64) string {
	const k = 1024
	if n < k {
		return formatInt(n) + " B"
	}
	v := float64(n) / float64(k)
	if v < k {
		return formatFloat1(v) + " KB"
	}
	v /= k
	if v < k {
		return formatFloat1(v) + " MB"
	}
	v /= k
	return formatFloat1(v) + " GB"
}

func formatInt(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := [24]byte{}
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func formatFloat1(v float64) string {
	scaled := int64(v*10 + 0.5)
	whole := scaled / 10
	frac := scaled % 10
	return formatInt(whole) + "." + string(byte('0'+frac))
}

// escapeKeyPath URL-escapes each path segment of an S3 key for use in app routes.
// Slashes are preserved as separators.
func escapeKeyPath(key string) string {
	parts := strings.Split(key, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return strings.Join(parts, "/")
}

// browseHref builds /b/{bucket}/browse/{prefix} (trailing slash preserved).
func browseHref(bucket, prefix string) string {
	if prefix == "" {
		return "/b/" + url.PathEscape(bucket) + "/browse/"
	}
	return "/b/" + url.PathEscape(bucket) + "/browse/" + escapeKeyPath(prefix)
}

func objectHref(bucket, key string) string {
	return "/b/" + url.PathEscape(bucket) + "/object/" + escapeKeyPath(key)
}

func previewHref(bucket, key string) string {
	return "/b/" + url.PathEscape(bucket) + "/preview/" + escapeKeyPath(key)
}

// buildCrumbs builds breadcrumb entries for the given bucket + prefix.
// The bucket crumb links to the bucket root; each segment links to its
// prefix browse URL.
func buildCrumbs(bucket, prefix string) []crumb {
	out := []crumb{{Label: bucket, Href: browseHref(bucket, "")}}
	if prefix == "" {
		return out
	}
	trimmed := strings.TrimSuffix(prefix, "/")
	parts := strings.Split(trimmed, "/")
	acc := ""
	for _, p := range parts {
		if p == "" {
			continue
		}
		acc += p + "/"
		out = append(out, crumb{Label: p, Href: browseHref(bucket, acc)})
	}
	return out
}

// parentPrefix returns the prefix one level up from the given prefix (which
// is expected to end in "/" or be empty). For object keys, use parentOfKey.
func parentPrefix(prefix string) string {
	trimmed := strings.TrimSuffix(prefix, "/")
	if trimmed == "" {
		return ""
	}
	i := strings.LastIndex(trimmed, "/")
	if i < 0 {
		return ""
	}
	return trimmed[:i+1]
}

func parentOfKey(key string) string {
	i := strings.LastIndex(key, "/")
	if i < 0 {
		return ""
	}
	return key[:i+1]
}

func nameOfKey(key string) string {
	i := strings.LastIndex(key, "/")
	if i < 0 {
		return key
	}
	return key[i+1:]
}

// parsePublicURLs parses a "bucket=url,bucket=url" string into a map.
func parsePublicURLs(raw string) map[string]string {
	out := map[string]string{}
	if strings.TrimSpace(raw) == "" {
		return out
	}
	for _, pair := range strings.Split(raw, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		eq := strings.IndexByte(pair, '=')
		if eq <= 0 {
			continue
		}
		k := strings.TrimSpace(pair[:eq])
		v := strings.TrimSpace(pair[eq+1:])
		if k != "" && v != "" {
			out[k] = v
		}
	}
	return out
}
