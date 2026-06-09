package main

import (
	"encoding/xml"
	"fmt"
	"time"
)

type atomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	XMLNS   string      `xml:"xmlns,attr"`
	Title   string      `xml:"title"`
	ID      string      `xml:"id"`
	Updated string      `xml:"updated"`
	Link    []atomLink  `xml:"link"`
	Entries []atomEntry `xml:"entry"`
}

type atomLink struct {
	Rel  string `xml:"rel,attr,omitempty"`
	Href string `xml:"href,attr"`
}

type atomEntry struct {
	Title   string     `xml:"title"`
	ID      string     `xml:"id"`
	Updated string     `xml:"updated"`
	Link    atomLink   `xml:"link"`
	Author  atomAuthor `xml:"author"`
	Summary string     `xml:"summary"`
}

type atomAuthor struct {
	Name  string `xml:"name"`
	Email string `xml:"email,omitempty"`
}

func buildAtomFeed(siteName, repoName, baseURL string, commits []CommitInfo) ([]byte, error) {
	feed := atomFeed{
		XMLNS:   "http://www.w3.org/2005/Atom",
		Title:   fmt.Sprintf("%s / %s — commits", siteName, repoName),
		ID:      baseURL + "/" + repoName,
		Updated: time.Now().UTC().Format(time.RFC3339),
		Link: []atomLink{
			{Href: baseURL + "/" + repoName},
			{Rel: "self", Href: baseURL + "/" + repoName + "/atom.xml"},
		},
	}
	if len(commits) > 0 {
		feed.Updated = commits[0].When.UTC().Format(time.RFC3339)
	}
	for _, c := range commits {
		feed.Entries = append(feed.Entries, atomEntry{
			Title:   c.Subject,
			ID:      baseURL + "/" + repoName + "/commit/" + c.SHA,
			Updated: c.When.UTC().Format(time.RFC3339),
			Link:    atomLink{Href: baseURL + "/" + repoName + "/commit/" + c.SHA},
			Author:  atomAuthor{Name: c.Author, Email: c.Email},
			Summary: c.Body,
		})
	}
	out, err := xml.MarshalIndent(feed, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), out...), nil
}
