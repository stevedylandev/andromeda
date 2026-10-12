package main

import (
	"bytes"
	"context"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

var betweenTags = regexp.MustCompile(`>\s+<`)
var spaces = regexp.MustCompile(`\s+`)

func normalizeHTML(s string) string {
	s = betweenTags.ReplaceAllString(s, "><")
	s = spaces.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, "> ", ">")
	s = strings.ReplaceAll(s, " <", "<")
	return strings.TrimSpace(s)
}

// The starters prefill the /admin/templates editor, so they must render the
// same markup as the templ defaults they stand in for.
func TestStartersMatchTemplDefaults(t *testing.T) {
	defaults := map[string]func(any) templ.Component{
		"index": func(v any) templ.Component { return indexBody(v.(indexView)) },
		"post":  func(v any) templ.Component { return postBody(v.(postView)) },
		"page":  func(v any) templ.Component { return pageBody(v.(pageView)) },
		"posts": func(v any) templ.Component { return postsBody(v.(postsView)) },
	}
	for _, k := range templateKinds {
		t.Run(k.Kind, func(t *testing.T) {
			v := sampleView(k.Kind)
			var want bytes.Buffer
			if err := defaults[k.Kind](v).Render(context.Background(), &want); err != nil {
				t.Fatal(err)
			}
			src, err := appFS.ReadFile("starters/" + k.Kind + ".html")
			if err != nil {
				t.Fatal(err)
			}
			tmpl, err := validateOverride(k.Kind, string(src))
			if err != nil {
				t.Fatal(err)
			}
			var got bytes.Buffer
			if err := tmpl.Execute(&got, v); err != nil {
				t.Fatal(err)
			}
			if g, w := normalizeHTML(got.String()), normalizeHTML(want.String()); g != w {
				t.Errorf("starter differs from templ default\nstarter: %s\ndefault: %s", g, w)
			}
		})
	}
}

func newTestApp(t *testing.T) *App {
	t.Helper()
	tmpls, err := buildTemplates()
	if err != nil {
		t.Fatal(err)
	}
	return &App{
		DB:        openPostsTestDB(t),
		Log:       slog.New(slog.NewTextHandler(io.Discard, nil)),
		Templates: tmpls,
		Overrides: &overrides{},
		SiteURL:   "https://example.com",
	}
}

func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
	return rec
}

func TestCustomTemplateOverridesBody(t *testing.T) {
	a := newTestApp(t)
	defer a.DB.Close()
	if _, err := createPost(a.DB, PostInput{Title: strp("Hi"), Slug: "hi", Content: "body", Status: "published", Lang: "en"}); err != nil {
		t.Fatal(err)
	}
	h := a.routes()

	rec := get(t, h, "/posts/hi")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `<h1>Hi</h1>`) {
		t.Fatalf("default render: %d %s", rec.Code, rec.Body.String())
	}

	// Invalid templates are rejected and leave the default in place.
	for _, src := range []string{"{{.Post.Nope}}", "{{if}}"} {
		rec = httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/admin/templates/post", strings.NewReader(url.Values{"template": {src}}.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		a.adminPostTemplate(rec, withKind(req, "post"))
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("%q: want 422, got %d", src, rec.Code)
		}
	}
	if a.Overrides.get("post") != nil {
		t.Fatal("invalid template was cached")
	}

	save := func(src string) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/admin/templates/post", strings.NewReader(url.Values{"template": {src}}.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		a.adminPostTemplate(rec, withKind(req, "post"))
		if rec.Code != http.StatusSeeOther {
			t.Fatalf("save: %d %s", rec.Code, rec.Body.String())
		}
	}
	save(`<h2 class="custom">{{.Post.TitleStr}} on {{.BlogTitle}}</h2>{{.RenderedContent}}`)
	body := get(t, h, "/posts/hi").Body.String()
	if !strings.Contains(body, `<h2 class="custom">Hi on My Blog</h2><p>body</p>`) {
		t.Fatalf("override not rendered: %s", body)
	}
	if !strings.Contains(body, `<meta property="og:title" content="Hi">`) {
		t.Fatal("layout meta missing with override")
	}

	// Overrides survive a restart.
	b := &App{DB: a.DB, Log: a.Log, Overrides: &overrides{}}
	b.loadOverrides()
	if b.Overrides.get("post") == nil {
		t.Fatal("override not loaded from settings")
	}

	// A runtime failure falls back to the default body.
	a.Overrides.set("post", template.Must(template.New("post").Parse(`{{index .Post.TagList 5}}`)))
	rec = get(t, h, "/posts/hi")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `<h1>Hi</h1>`) {
		t.Fatalf("fallback: %d %s", rec.Code, rec.Body.String())
	}

	// Clearing removes the override.
	save("")
	if a.Overrides.get("post") != nil {
		t.Fatal("override not cleared")
	}
	if v, _ := getSetting(a.DB, templateSettingKey("post")); v != "" {
		t.Fatalf("setting not cleared: %q", v)
	}
}

func withKind(r *http.Request, kind string) *http.Request {
	r.SetPathValue("kind", kind)
	return r
}

func TestPublicPagesRender(t *testing.T) {
	a := newTestApp(t)
	defer a.DB.Close()
	if _, err := createPost(a.DB, PostInput{Title: strp("<b>x</b>"), Slug: "x", Content: "body", Status: "published", Lang: "en", Tags: strp("a")}); err != nil {
		t.Fatal(err)
	}
	if _, err := createPage(a.DB, "About", "about", "about me", true, 0); err != nil {
		t.Fatal(err)
	}
	_ = setSetting(a.DB, "intro_content", "hi\n\n{{latest_posts}}")
	h := a.routes()
	for path, want := range map[string]string{
		"/":        `<span class="tag">a</span>`,
		"/posts":   `<h1>Posts</h1>`,
		"/posts/x": `<h1>&lt;b&gt;x&lt;/b&gt;</h1>`,
		"/about":   `<h1>About</h1>`,
	} {
		rec := get(t, h, path)
		if rec.Code != 200 || !strings.Contains(rec.Body.String(), want) {
			t.Errorf("%s: %d, missing %q in %s", path, rec.Code, want, rec.Body.String())
		}
		if strings.Contains(rec.Body.String(), "<b>x</b>") {
			t.Errorf("%s: unescaped title", path)
		}
	}
}
