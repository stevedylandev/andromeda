package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const userAgent = "Mozilla/5.0 (compatible; OGPreview/1.0)"

type ogResult struct {
	OGTags   map[string]string
	OGOrder  []string
	Favicon  string
	LinkTags []linkTag
}

func fetchOGData(ctx context.Context, target string) (*ogResult, error) {
	parsed, err := url.Parse(target)
	if err != nil {
		return nil, fmt.Errorf("Invalid URL: %w", err)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to create HTTP client: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP error: %s", resp.Status)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") && !strings.Contains(ct, "application/xhtml") {
		return nil, fmt.Errorf("Not an HTML page (Content-Type: %s)", ct)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("Failed to read response body: %w", err)
	}
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, errors.New("Failed to parse HTML")
	}

	res := &ogResult{OGTags: map[string]string{}}
	walk(doc, func(n *html.Node) {
		if n.Type != html.ElementNode {
			return
		}
		if strings.EqualFold(n.Data, "meta") {
			attrs := attrsOf(n)
			key := attrs["property"]
			if key == "" {
				key = attrs["name"]
			}
			if strings.HasPrefix(key, "og:") {
				if _, exists := res.OGTags[key]; !exists {
					res.OGTags[key] = attrs["content"]
					res.OGOrder = append(res.OGOrder, key)
				}
			}
		}
	})

	if image, ok := res.OGTags["og:image"]; ok {
		if u, err := parsed.Parse(image); err == nil {
			res.OGTags["og:image"] = u.String()
		}
	}

	res.Favicon = extractFavicon(doc, parsed)
	res.LinkTags = extractLinkTags(doc, parsed)
	return res, nil
}

func extractFavicon(doc *html.Node, base *url.URL) string {
	rels := []string{"icon", "shortcut icon", "apple-touch-icon"}
	for _, want := range rels {
		var found string
		walk(doc, func(n *html.Node) {
			if found != "" || n.Type != html.ElementNode || !strings.EqualFold(n.Data, "link") {
				return
			}
			attrs := attrsOf(n)
			if strings.EqualFold(strings.TrimSpace(attrs["rel"]), want) {
				if href := attrs["href"]; href != "" {
					if u, err := base.Parse(href); err == nil {
						found = u.String()
					}
				}
			}
		})
		if found != "" {
			return found
		}
	}
	if fb, err := base.Parse("/favicon.ico"); err == nil {
		return fb.String()
	}
	return ""
}

func extractLinkTags(doc *html.Node, base *url.URL) []linkTag {
	var head *html.Node
	walk(doc, func(n *html.Node) {
		if head != nil {
			return
		}
		if n.Type == html.ElementNode && strings.EqualFold(n.Data, "head") {
			head = n
		}
	})
	if head == nil {
		return nil
	}
	var out []linkTag
	walk(head, func(n *html.Node) {
		if n.Type != html.ElementNode || !strings.EqualFold(n.Data, "link") {
			return
		}
		attrs := attrsOf(n)
		href := attrs["href"]
		if href != "" {
			if u, err := base.Parse(href); err == nil {
				href = u.String()
			}
		}
		extras := []string{}
		for _, a := range n.Attr {
			if a.Key == "rel" || a.Key == "href" {
				continue
			}
			extras = append(extras, fmt.Sprintf(`%s="%s"`, a.Key, a.Val))
		}
		out = append(out, linkTag{Rel: attrs["rel"], Href: href, Extra: strings.Join(extras, " ")})
	})
	return out
}

func attrsOf(n *html.Node) map[string]string {
	out := make(map[string]string, len(n.Attr))
	for _, a := range n.Attr {
		out[strings.ToLower(a.Key)] = a.Val
	}
	return out
}

func walk(n *html.Node, visit func(*html.Node)) {
	visit(n)
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walk(c, visit)
	}
}
