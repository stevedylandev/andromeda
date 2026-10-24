package main

//go:generate go run github.com/a-h/templ/cmd/templ@v0.3.1001 generate

import (
	"bytes"
	"html/template"
	"net/http"
	"sync"

	"github.com/a-h/templ"
)

// View models for the public pages. They are passed both to the templ
// components and to custom body templates, so their fields (including the
// promoted siteContext ones) are the contract documented on /admin/templates.

type indexView struct {
	siteContext
	BlogDescription string
	IntroHTML       template.HTML
	Posts           []Post
}

type postView struct {
	siteContext
	Post            Post
	RenderedContent template.HTML
	Weather         Weather
}

type pageView struct {
	siteContext
	Page            Page
	RenderedContent template.HTML
}

type postsView struct {
	siteContext
	Posts []Post
}

// templateKinds are the public views whose body can be replaced from
// /admin/templates. Each is stored in the "template_<kind>" setting.
var templateKinds = []struct{ Kind, Label string }{
	{"index", "home"},
	{"post", "post"},
	{"page", "page"},
	{"posts", "posts list"},
}

func isTemplateKind(kind string) bool {
	for _, k := range templateKinds {
		if k.Kind == kind {
			return true
		}
	}
	return false
}

func templateSettingKey(kind string) string { return "template_" + kind }

// overrides caches parsed custom body templates by kind.
type overrides struct {
	mu    sync.RWMutex
	tmpls map[string]*template.Template
}

func (o *overrides) get(kind string) *template.Template {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.tmpls[kind]
}

func (o *overrides) set(kind string, t *template.Template) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.tmpls == nil {
		o.tmpls = map[string]*template.Template{}
	}
	if t == nil {
		delete(o.tmpls, kind)
		return
	}
	o.tmpls[kind] = t
}

func (a *App) loadOverrides() {
	for _, k := range templateKinds {
		src, _ := getSetting(a.DB, templateSettingKey(k.Kind))
		if src == "" {
			continue
		}
		t, err := template.New(k.Kind).Parse(src)
		if err != nil {
			a.Log.Error("custom template parse failed", "kind", k.Kind, "err", err)
			continue
		}
		a.Overrides.set(k.Kind, t)
	}
}

// validateOverride parses src and executes it against sample data, so errors
// like unknown fields surface on save rather than on the live site.
func validateOverride(kind, src string) (*template.Template, error) {
	t, err := template.New(kind).Parse(src)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, sampleView(kind)); err != nil {
		return nil, err
	}
	return t, nil
}

// body returns the custom body for kind if one is set and renders cleanly,
// otherwise the default templ component.
func (a *App) body(kind string, data any, def templ.Component) templ.Component {
	t := a.Overrides.get(kind)
	if t == nil {
		return def
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		a.Log.Error("custom template render failed, using default", "kind", kind, "err", err)
		return def
	}
	return templ.Raw(buf.String())
}

func (a *App) renderPublic(w http.ResponseWriter, r *http.Request, site siteContext, title string, meta, body templ.Component) {
	var buf bytes.Buffer
	if err := layout(site, title, meta, body).Render(r.Context(), &buf); err != nil {
		a.Log.Error("render failed", "err", err)
		http.Error(w, "render error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = buf.WriteTo(w)
}

// sampleView returns representative data for kind, used to validate custom
// templates and to check the starters match the templ defaults.
func sampleView(kind string) any {
	s := func(v string) *string { return &v }
	site := siteContext{
		BlogTitle: "My Blog",
		NavLinks:  []NavLink{{Label: "about", URL: "/about"}},
		SiteURL:   "https://example.com",
	}
	posts := []Post{
		{
			Slug: "hello-world", Title: s("Hello world"), PublishedDate: s("2026-01-02"),
			MetaDescription: s("A first post"), Tags: s("go, templ"), Weather: s("{}"),
			Content: "Hello **world**", Status: "published",
		},
		{Slug: "untitled", Content: "A post with no title that runs on for a while", Status: "published"},
	}
	switch kind {
	case "index":
		return indexView{siteContext: site, BlogDescription: "A blog", IntroHTML: "<p>Welcome</p>", Posts: posts}
	case "post":
		return postView{
			siteContext: site, Post: posts[0], RenderedContent: "<p>Hello <strong>world</strong></p>",
			Weather: Weather{Icon: "<svg></svg>", Conditions: "Sunny", Temperature: "72°F", Location: "Somewhere, OK"},
		}
	case "page":
		return pageView{siteContext: site, Page: Page{Title: "About", Slug: "about", IsPublished: true}, RenderedContent: "<p>About me</p>"}
	case "posts":
		return postsView{siteContext: site, Posts: posts}
	}
	return nil
}
