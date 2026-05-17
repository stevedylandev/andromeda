package main

import (
	"net/url"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func parseDoc(t *testing.T, s string) *html.Node {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(s))
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestFaviconExtractionPriorityAndFallback(t *testing.T) {
	base, _ := url.Parse("https://example.com/path/page")
	cases := []struct{ name, html, want string }{
		{"icon", `<head><link rel="icon" href="/icon.png"></head>`, "https://example.com/icon.png"},
		{"shortcut", `<head><link rel="shortcut icon" href="shortcut.ico"></head>`, "https://example.com/path/shortcut.ico"},
		{"apple", `<head><link rel="apple-touch-icon" href="https://cdn.test/apple.png"></head>`, "https://cdn.test/apple.png"},
		{"priority", `<head><link rel="apple-touch-icon" href="/apple.png"><link rel="shortcut icon" href="/shortcut.ico"><link rel="icon" href="/icon.png"></head>`, "https://example.com/icon.png"},
		{"fallback", `<head></head>`, "https://example.com/favicon.ico"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractFavicon(parseDoc(t, tc.html), base); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestExtractLinkTags(t *testing.T) {
	base, _ := url.Parse("https://example.com/dir/page")
	doc := parseDoc(t, `<html><head><link rel="preload" href="/style.css" as="style"><link rel="canonical" href="canonical"></head><body><link rel="ignored" href="/body"></body></html>`)
	links := extractLinkTags(doc, base)
	if len(links) != 2 {
		t.Fatalf("links %#v", links)
	}
	if links[0].Rel != "preload" || links[0].Href != "https://example.com/style.css" || links[0].Extra != `as="style"` {
		t.Fatalf("first link %#v", links[0])
	}
	if links[1].Href != "https://example.com/dir/canonical" {
		t.Fatalf("relative href %q", links[1].Href)
	}
	if got := extractLinkTags(parseDoc(t, `<html><body></body></html>`), base); got != nil {
		t.Fatalf("expected nil without head, got %#v", got)
	}
}
