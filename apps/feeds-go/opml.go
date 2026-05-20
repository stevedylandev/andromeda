package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type OPMLEntry struct {
	XMLURL   string
	Title    string
	HTMLURL  string
	Category string
}

func parseOPML(content string) []OPMLEntry {
	dec := xml.NewDecoder(strings.NewReader(content))
	type outline struct {
		Title   string    `xml:"title,attr"`
		Text    string    `xml:"text,attr"`
		XMLURL  string    `xml:"xmlUrl,attr"`
		HTMLURL string    `xml:"htmlUrl,attr"`
		Nodes   []outline `xml:"outline"`
	}
	type opml struct {
		Body struct {
			Nodes []outline `xml:"outline"`
		} `xml:"body"`
	}
	var doc opml
	if err := dec.Decode(&doc); err != nil {
		return nil
	}
	var out []OPMLEntry
	var walk func(nodes []outline, category string)
	walk = func(nodes []outline, category string) {
		for _, node := range nodes {
			title := firstNonEmpty(node.Title, node.Text)
			if strings.TrimSpace(node.XMLURL) != "" {
				out = append(out, OPMLEntry{XMLURL: strings.TrimSpace(node.XMLURL), Title: title, HTMLURL: strings.TrimSpace(node.HTMLURL), Category: strings.TrimSpace(category)})
				if len(node.Nodes) > 0 {
					walk(node.Nodes, title)
				}
				continue
			}
			walk(node.Nodes, title)
		}
	}
	walk(doc.Body.Nodes, "")
	return out
}

func fetchOPMLDoc(ctx context.Context, opmlURL string) (string, error) {
	client := buildHTTPClient()
	req, err := newRequest(ctx, http.MethodGet, opmlURL)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("upstream returned %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", fmt.Errorf("read failed: %w", err)
	}
	return string(body), nil
}
