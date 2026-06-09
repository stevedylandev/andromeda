package main

import (
	"embed"
	"html/template"
	"log/slog"
	"time"
)

//go:embed templates/*.html static/* static/fonts/*
var appFS embed.FS

type App struct {
	Log          *slog.Logger
	Templates    map[string]*template.Template
	RepoRoot     string
	SiteName     string
	BaseURL      string
	CloneBaseURL string
	CloneSSHHost string
}

type pageBase struct {
	SiteName string
	RepoName string
	BaseURL  string
}

type indexPageData struct {
	pageBase
	Repos []RepoSummary
}

type repoPageData struct {
	pageBase
	Repo          RepoSummary
	Ref           string
	ReadmeHTML    template.HTML
	HasReadme     bool
	Branches      []RefInfo
	Tags          []RefInfo
	Commits       []CommitInfo
	DefaultRef    string
	LatestCommit  CommitInfo
	HasLatest     bool
	Entries       []TreeEntry
	CloneHTTPSURL string
	CloneSSHURL   string
}

type treePageData struct {
	pageBase
	Repo        RepoSummary
	Ref         string
	DefaultRef  string
	Path        string
	Breadcrumbs []Breadcrumb
	Entries     []TreeEntry
}

type blobPageData struct {
	pageBase
	Repo            RepoSummary
	Ref             string
	DefaultRef      string
	Path            string
	Breadcrumbs     []Breadcrumb
	Binary          bool
	Size            int64
	HighlightedHTML template.HTML
}

type logPageData struct {
	pageBase
	Repo       RepoSummary
	Ref        string
	DefaultRef string
	Commits    []CommitInfo
	Page       int
	NextPage   int
	PrevPage   int
	HasNext    bool
	HasPrev    bool
}

type commitPageData struct {
	pageBase
	Repo       RepoSummary
	DefaultRef string
	Commit     CommitInfo
	Files      []FilePatch
	Stats      DiffStats
}

type refsPageData struct {
	pageBase
	Repo       RepoSummary
	DefaultRef string
	Branches   []RefInfo
	Tags       []RefInfo
}

type Breadcrumb struct {
	Name string
	Href string
}

type RepoSummary struct {
	Name        string
	Description string
	DefaultRef  string
	LastCommit  time.Time
}

type RefInfo struct {
	Name   string
	SHA    string
	Time   time.Time
	Author string
}

type CommitInfo struct {
	SHA       string
	ShortSHA  string
	Author    string
	Email     string
	When      time.Time
	Subject   string
	Body      string
	ParentSHA string
}

type TreeEntry struct {
	Name   string
	Path   string
	IsDir  bool
	Size   int64
	Mode   string
	Commit CommitInfo
}

type FilePatch struct {
	From    string
	To      string
	IsBin   bool
	Hunks   []DiffHunk
	Added   int
	Removed int
}

type DiffHunk struct {
	Header string
	Lines  []DiffLine
}

type DiffLine struct {
	Kind   string // "ctx", "add", "del"
	Text   string
	OldNum int
	NewNum int
}

type DiffStats struct {
	Files   int
	Added   int
	Removed int
}
