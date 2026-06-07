package main

import (
	"database/sql"
	"embed"
	"html/template"
	"log/slog"
	"strings"

	poststorage "github.com/stevedylandev/andromeda/apps/posts/storage"
	"github.com/stevedylandev/andromeda/pkg/auth"
)

//go:embed templates/*.html static/*
var appFS embed.FS

type App struct {
	DB           *sql.DB
	Log          *slog.Logger
	Templates    map[string]*template.Template
	Sessions     *auth.Store
	AppPassword  string
	CookieSecure bool
	UploadsDir   string
	Storage      poststorage.Backend
	SiteURL      string
}

type Post struct {
	ID              int64
	ShortID         string
	Title           *string
	Slug            string
	Alias           *string
	CanonicalURL    *string
	PublishedDate   *string
	MetaDescription *string
	MetaImage       *string
	Lang            string
	Tags            *string
	Weather         *string
	Content         string
	Status          string
	CreatedAt       string
	UpdatedAt       string
}

func (p Post) DisplayTitle() string {
	if p.Title != nil {
		if t := strings.TrimSpace(*p.Title); t != "" {
			return t
		}
	}
	body := strings.ReplaceAll(strings.ReplaceAll(p.Content, "\n", ""), "\r", "")
	runes := []rune(body)
	n := 25
	if len(runes) < n {
		n = len(runes)
	}
	snip := strings.TrimSpace(string(runes[:n]))
	if snip == "" {
		return "Untitled"
	}
	if len([]rune(p.Content)) > 60 {
		return snip + "…"
	}
	return snip
}

func (p Post) TitleStr() string {
	if p.Title != nil {
		return *p.Title
	}
	return ""
}

func (p Post) HasTitle() bool {
	return p.Title != nil && strings.TrimSpace(*p.Title) != ""
}

func (p Post) PublishedDateStr() string {
	if p.PublishedDate != nil {
		return *p.PublishedDate
	}
	return ""
}

func (p Post) AliasStr() string {
	if p.Alias != nil {
		return *p.Alias
	}
	return ""
}

func (p Post) MetaDescriptionStr() string {
	if p.MetaDescription != nil {
		return *p.MetaDescription
	}
	return ""
}

func (p Post) MetaImageStr() string {
	if p.MetaImage != nil {
		return *p.MetaImage
	}
	return ""
}

func (p Post) CanonicalURLStr() string {
	if p.CanonicalURL != nil {
		return *p.CanonicalURL
	}
	return ""
}

func (p Post) TagsStr() string {
	if p.Tags != nil {
		return *p.Tags
	}
	return ""
}

func (p Post) TagList() []string {
	if p.Tags == nil {
		return nil
	}
	var out []string
	for _, t := range strings.Split(*p.Tags, ",") {
		if v := strings.TrimSpace(t); v != "" {
			out = append(out, v)
		}
	}
	return out
}

type Page struct {
	ID          int64
	ShortID     string
	Title       string
	Slug        string
	Content     string
	IsPublished bool
	NavOrder    int64
	CreatedAt   string
	UpdatedAt   string
}

type UploadedFile struct {
	ID             int64
	ShortID        string
	Filename       string
	OriginalName   string
	ContentType    string
	Size           int64
	CreatedAt      string
	StorageBackend string
}

func (f UploadedFile) SizeHuman() string {
	return humanSize(f.Size)
}

func (f UploadedFile) IsImage() bool {
	return strings.HasPrefix(f.ContentType, "image/")
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

type NavLink struct {
	Label string
	URL   string
}

type siteContext struct {
	BlogTitle  string
	NavLinks   []NavLink
	FaviconURL string
	OGImageURL string
	SiteURL    string
	HeaderHTML template.HTML
	FooterHTML template.HTML
}

type loginPageData struct {
	Error string
}

type indexPageData struct {
	BlogTitle       string
	BlogDescription string
	IntroHTML       template.HTML
	Posts           []Post
	NavLinks        []NavLink
	FaviconURL      string
	OGImageURL      string
	SiteURL         string
	HeaderHTML      template.HTML
	FooterHTML      template.HTML
}

type postPageData struct {
	BlogTitle       string
	NavLinks        []NavLink
	Post            Post
	RenderedContent template.HTML
	FaviconURL      string
	OGImageURL      string
	SiteURL         string
	HeaderHTML      template.HTML
	FooterHTML      template.HTML
}

type pagePageData struct {
	BlogTitle       string
	NavLinks        []NavLink
	Page            Page
	RenderedContent template.HTML
	FaviconURL      string
	OGImageURL      string
	SiteURL         string
	HeaderHTML      template.HTML
	FooterHTML      template.HTML
}

type postsListPageData struct {
	BlogTitle  string
	NavLinks   []NavLink
	Posts      []Post
	FaviconURL string
	OGImageURL string
	SiteURL    string
	HeaderHTML template.HTML
	FooterHTML template.HTML
}

type adminIndexPageData struct {
	Posts []Post
}

type adminPostFormPageData struct {
	Post  *Post
	Error string
}

type adminPagesPageData struct {
	Pages []Page
}

type adminPageFormPageData struct {
	Page  *Page
	Error string
}

type adminSettingsPageData struct {
	BlogTitle       string
	BlogDescription string
	IntroContent    string
	NavLinksRaw     string
	CustomCSS       string
	DefaultCSS      string
	FaviconURL      string
	OGImageURL      string
	CustomHeader    string
	CustomFooter    string
	DefaultLocation string
	Success         bool
}

type adminFilesPageData struct {
	Files   []UploadedFile
	SiteURL string
	Error   string
	Success bool
}

type adminImportPageData struct {
	Error    string
	Imported *int
	Skipped  *int
}
