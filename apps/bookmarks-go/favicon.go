package main

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const faviconUA = "andromeda-bookmarks-go/0.1 (+https://github.com/stevedylandev/andromeda)"

func discoverFavicon(ctx context.Context, pageURL string) string {
	parsed, err := url.Parse(pageURL)
	if err != nil {
		return ""
	}
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", faviconUA)
	if resp, err := client.Do(req); err == nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if href := findFaviconHref(string(body)); href != "" {
			if u, err := parsed.Parse(href); err == nil {
				return u.String()
			}
		}
	}
	if u, err := parsed.Parse("/favicon.ico"); err == nil {
		return u.String()
	}
	return ""
}

func findFaviconHref(doc string) string {
	node, err := html.Parse(strings.NewReader(doc))
	if err != nil {
		return ""
	}
	wants := []string{"icon", "shortcut icon", "apple-touch-icon"}
	var found string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if found != "" {
			return
		}
		if n.Type == html.ElementNode && strings.EqualFold(n.Data, "link") {
			rel, href := "", ""
			for _, a := range n.Attr {
				switch strings.ToLower(a.Key) {
				case "rel":
					rel = strings.ToLower(strings.TrimSpace(a.Val))
				case "href":
					href = a.Val
				}
			}
			for _, want := range wants {
				if rel == want {
					if href != "" {
						found = href
					}
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(node)
	return found
}
