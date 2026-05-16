package main

import (
	"encoding/xml"
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
