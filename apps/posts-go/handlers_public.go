package main

import (
	"html/template"
	"net/http"
	"strings"
)

func (a *App) site() siteContext {
	blogTitle, _ := getSetting(a.DB, "blog_title")
	if blogTitle == "" {
		blogTitle = "My Blog"
	}
	navLinksRaw, _ := getSetting(a.DB, "nav_links")
	favicon, _ := getSetting(a.DB, "favicon_url")
	ogImage, _ := getSetting(a.DB, "og_image_url")
	header, _ := getSetting(a.DB, "custom_header")
	footer, _ := getSetting(a.DB, "custom_footer")
	return siteContext{
		BlogTitle:  blogTitle,
		NavLinks:   parseNavLinks(navLinksRaw),
		FaviconURL: favicon,
		OGImageURL: ogImage,
		SiteURL:    a.SiteURL,
		HeaderHTML: template.HTML(renderMarkdown(header)),
		FooterHTML: template.HTML(renderMarkdown(footer)),
	}
}

func renderLatestPostsEmbed(posts []Post) string {
	var b strings.Builder
	b.WriteString(`<div class="post-list">`)
	for i := range posts {
		p := &posts[i]
		b.WriteString(`<a href="/posts/` + p.Slug + `" class="post-item"><div class="post-item-info"><span class="post-title">` + p.DisplayTitle() + `</span>`)
		if p.Tags != nil && *p.Tags != "" {
			b.WriteString(`<span class="post-tags">`)
			for _, t := range strings.Split(*p.Tags, ",") {
				if v := strings.TrimSpace(t); v != "" {
					b.WriteString(`<span class="tag">` + v + `</span>`)
				}
			}
			b.WriteString(`</span>`)
		}
		b.WriteString(`</div>`)
		if p.PublishedDate != nil {
			b.WriteString(`<time class="post-date">` + *p.PublishedDate + `</time>`)
		}
		b.WriteString(`</a>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}

func (a *App) publicIndex(w http.ResponseWriter, r *http.Request) {
	ctx := a.site()
	blogDesc, _ := getSetting(a.DB, "blog_description")
	intro, _ := getSetting(a.DB, "intro_content")

	posts, err := getPublishedPosts(a.DB, 0)
	if err != nil {
		a.Log.Error("list posts", "err", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	introHTML := renderMarkdown(intro)
	if strings.Contains(intro, "{{latest_posts}}") {
		take := len(posts)
		if take > 5 {
			take = 5
		}
		embed := renderLatestPostsEmbed(posts[:take])
		introHTML = strings.ReplaceAll(introHTML, "<p>{{latest_posts}}</p>", embed)
		introHTML = strings.ReplaceAll(introHTML, "{{latest_posts}}", embed)
	}

	a.renderPage(w, "index.html", indexPageData{
		BlogTitle: ctx.BlogTitle, BlogDescription: blogDesc,
		IntroHTML: template.HTML(introHTML),
		Posts:     posts,
		NavLinks:  ctx.NavLinks, FaviconURL: ctx.FaviconURL, OGImageURL: ctx.OGImageURL,
		SiteURL: ctx.SiteURL, HeaderHTML: ctx.HeaderHTML, FooterHTML: ctx.FooterHTML,
	})
}

func (a *App) publicPost(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	post, err := getPostBySlug(a.DB, slug)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	if post == nil || post.Status != "published" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	ctx := a.site()
	rendered := renderMarkdown(post.Content)
	a.renderPage(w, "post.html", postPageData{
		BlogTitle: ctx.BlogTitle, NavLinks: ctx.NavLinks, Post: *post,
		RenderedContent: template.HTML(rendered),
		FaviconURL:      ctx.FaviconURL, OGImageURL: ctx.OGImageURL,
		SiteURL: ctx.SiteURL, HeaderHTML: ctx.HeaderHTML, FooterHTML: ctx.FooterHTML,
	})
}

func (a *App) publicPage(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	page, err := getPageBySlug(a.DB, slug)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	if page == nil || !page.IsPublished {
		// fallback: alias redirect or 404
		if redirect, err := findAliasRedirect(a.DB, slug); err == nil && redirect != "" {
			http.Redirect(w, r, redirect, http.StatusMovedPermanently)
			return
		}
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	ctx := a.site()
	rendered := renderMarkdown(page.Content)
	a.renderPage(w, "page.html", pagePageData{
		BlogTitle: ctx.BlogTitle, NavLinks: ctx.NavLinks, Page: *page,
		RenderedContent: template.HTML(rendered),
		FaviconURL:      ctx.FaviconURL, OGImageURL: ctx.OGImageURL,
		SiteURL: ctx.SiteURL, HeaderHTML: ctx.HeaderHTML, FooterHTML: ctx.FooterHTML,
	})
}

func (a *App) publicPostsList(w http.ResponseWriter, r *http.Request) {
	ctx := a.site()
	posts, err := getPublishedPosts(a.DB, 0)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	a.renderPage(w, "posts.html", postsListPageData{
		BlogTitle: ctx.BlogTitle, NavLinks: ctx.NavLinks, Posts: posts,
		FaviconURL: ctx.FaviconURL, OGImageURL: ctx.OGImageURL,
		SiteURL: ctx.SiteURL, HeaderHTML: ctx.HeaderHTML, FooterHTML: ctx.FooterHTML,
	})
}

func (a *App) customCSS(w http.ResponseWriter, r *http.Request) {
	css, _ := getSetting(a.DB, "custom_css")
	w.Header().Set("Content-Type", "text/css")
	_, _ = w.Write([]byte(css))
}

func (a *App) serveUploadedFile(w http.ResponseWriter, r *http.Request) {
	filename := r.PathValue("filename")
	if strings.Contains(filename, "..") || strings.ContainsAny(filename, "/\\") {
		http.NotFound(w, r)
		return
	}
	f, _ := getFileByFilename(a.DB, filename)
	if f != nil && f.StorageBackend == "r2" {
		if a.Storage != nil && a.Storage.Name() == "r2" {
			if u := a.Storage.PublicURL(filename); u != "" {
				http.Redirect(w, r, u, http.StatusTemporaryRedirect)
				return
			}
		}
		http.NotFound(w, r)
		return
	}
	path := a.UploadsDir + "/" + filename
	data, err := readFile(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	ct := mimeFromPath(filename)
	if f != nil && f.ContentType != "" {
		ct = f.ContentType
	}
	w.Header().Set("Content-Type", ct)
	_, _ = w.Write(data)
}

func (a *App) rssFeed(w http.ResponseWriter, r *http.Request) {
	blogTitle, _ := getSetting(a.DB, "blog_title")
	if blogTitle == "" {
		blogTitle = "My Blog"
	}
	blogDesc, _ := getSetting(a.DB, "blog_description")
	posts, err := getPublishedPosts(a.DB, 0)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	var items strings.Builder
	for _, p := range posts {
		link := a.SiteURL + "/posts/" + xmlEscape(p.Slug)
		title := ""
		if p.Title != nil {
			if t := strings.TrimSpace(*p.Title); t != "" {
				title = xmlEscape(t)
			}
		}
		desc := ""
		if p.MetaDescription != nil && *p.MetaDescription != "" {
			desc = xmlEscape(*p.MetaDescription)
		} else {
			runes := []rune(p.Content)
			n := 200
			if len(runes) < n {
				n = len(runes)
			}
			desc = xmlEscape(string(runes[:n]))
		}
		rawDate := p.CreatedAt
		if p.PublishedDate != nil {
			rawDate = *p.PublishedDate
		}
		pubDate := toRFC2822(rawDate)
		items.WriteString("    <item>\n      <title>" + title + "</title>\n      <link>" + link + "</link>\n      <guid>" + link + "</guid>\n      <description>" + desc + "</description>\n      <pubDate>" + pubDate + "</pubDate>\n    </item>\n")
	}
	lastBuild := ""
	if len(posts) > 0 && posts[0].PublishedDate != nil {
		lastBuild = toRFC2822(*posts[0].PublishedDate)
	}
	out := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
  <channel>
    <title>` + xmlEscape(blogTitle) + `</title>
    <link>` + a.SiteURL + `</link>
    <description>` + xmlEscape(blogDesc) + `</description>
    <lastBuildDate>` + lastBuild + `</lastBuildDate>
    <atom:link href="` + a.SiteURL + `/feed.xml" rel="self" type="application/rss+xml"/>
` + items.String() + `  </channel>
</rss>`
	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	_, _ = w.Write([]byte(out))
}

func readFile(path string) ([]byte, error) {
	return readFileImpl(path)
}
