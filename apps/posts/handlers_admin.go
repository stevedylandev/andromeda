package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/stevedylandev/andromeda/pkg/auth"
)

const importMaxBytes = 50 * 1024 * 1024
const uploadMaxBytes = 10 * 1024 * 1024
const bodyLimit = 51 * 1024 * 1024

func (a *App) loginGet(w http.ResponseWriter, r *http.Request) {
	a.renderPage(w, "login.html", loginPageData{Error: r.URL.Query().Get("error")})
}

func (a *App) loginPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/admin/login?error=Bad+request", http.StatusSeeOther)
		return
	}
	if !auth.VerifyPassword(r.FormValue("password"), a.AppPassword) {
		http.Redirect(w, r, "/admin/login?error=Invalid+password", http.StatusSeeOther)
		return
	}
	token, err := a.Sessions.Create()
	if err != nil {
		a.Log.Error("create session", "err", err)
		http.Redirect(w, r, "/admin/login?error=Server+error", http.StatusSeeOther)
		return
	}
	a.Sessions.PruneExpired()
	http.SetCookie(w, a.Sessions.SessionCookie(token))
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(a.Sessions.CookieName); err == nil && c.Value != "" {
		a.Sessions.Delete(c.Value)
	}
	http.SetCookie(w, a.Sessions.ClearCookie())
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

func (a *App) adminIndex(w http.ResponseWriter, r *http.Request) {
	posts, err := getAllPosts(a.DB)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	a.renderPage(w, "admin_index.html", adminIndexPageData{Posts: posts})
}

func (a *App) adminNewPost(w http.ResponseWriter, r *http.Request) {
	a.renderPage(w, "admin_post_form.html", adminPostFormPageData{Error: r.URL.Query().Get("error")})
}

func (a *App) adminCreatePost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/admin/posts/new?error=Bad+request", http.StatusSeeOther)
		return
	}
	attrs := parseAttributes(r.FormValue("attributes"))
	title := strings.TrimSpace(attrs.Title)
	slug := deriveSlugWith(a, title, strings.TrimSpace(attrs.Slug))
	status := "draft"
	if r.FormValue("action") == "publish" {
		status = "published"
	}
	lang := "en"
	if l := strings.TrimSpace(attrs.Lang); l != "" {
		lang = l
	}
	defaultLocation, err := getSetting(a.DB, "default_location")
	if err != nil {
		defaultLocation = ""
	}
	weather := getWeather(defaultLocation)
	if newWeather := strings.TrimSpace(attrs.Weather); newWeather != "" {
		weather = newWeather
	}
	pub := strings.TrimSpace(attrs.PublishedDate)
	if pub == "" {
		pub = nowDatetime()
	}
	in := PostInput{
		Title: optStr(title), Slug: slug, Content: r.FormValue("content"),
		Status: status, Alias: optStr(attrs.Alias),
		PublishedDate:   &pub,
		MetaDescription: optStr(attrs.MetaDescription),
		MetaImage:       optStr(attrs.MetaImage),
		Lang:            lang, Tags: optStr(attrs.Tags),
		Weather: 				 optStr(weather),
	}
	if _, err := createPost(a.DB, in); err != nil {
		a.Log.Error("create post", "err", err)
		http.Redirect(w, r, "/admin/posts/new?error=Failed+to+create+post", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func deriveSlugWith(a *App, title, slug string) string {
	if slug != "" {
		return slug
	}
	if s := slugify(title); s != "" {
		return s
	}
	id, _ := auth.GenerateShortID(10)
	return id
}

func (a *App) adminEditPost(w http.ResponseWriter, r *http.Request) {
	shortID := r.PathValue("id")
	post, err := getPostByShortID(a.DB, shortID)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	if post == nil {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}
	a.renderPage(w, "admin_post_form.html", adminPostFormPageData{Post: post, Error: r.URL.Query().Get("error")})
}

func (a *App) adminUpdatePost(w http.ResponseWriter, r *http.Request) {
	shortID := r.PathValue("id")
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/admin/posts/"+shortID+"/edit?error=Bad+request", http.StatusSeeOther)
		return
	}
	attrs := parseAttributes(r.FormValue("attributes"))
	title := strings.TrimSpace(attrs.Title)
	slug := deriveSlugWith(a, title, strings.TrimSpace(attrs.Slug))
	status := "draft"
	if r.FormValue("action") == "publish" {
		status = "published"
	}
	lang := "en"
	if l := strings.TrimSpace(attrs.Lang); l != "" {
		lang = l
	}
	var pubDate *string
	if t := strings.TrimSpace(attrs.PublishedDate); t != "" {
		pubDate = &t
	}
		defaultLocation, err := getSetting(a.DB, "default_location")
	if err != nil {
		defaultLocation = ""
	}
	weather := getWeather(defaultLocation)
	if newWeather := strings.TrimSpace(attrs.Weather); newWeather != "" {
		weather = newWeather
	}
	in := PostInput{
		Title: optStr(title), Slug: slug, Content: r.FormValue("content"),
		Status: status, Alias: optStr(attrs.Alias),
		PublishedDate:   pubDate,
		MetaDescription: optStr(attrs.MetaDescription),
		MetaImage:       optStr(attrs.MetaImage),
		Lang:            lang, Tags: optStr(attrs.Tags),
		Weather:				 optStr(weather),
	}
	if _, err := updatePost(a.DB, shortID, in); err != nil {
		a.Log.Error("update post", "err", err)
		http.Redirect(w, r, "/admin/posts/"+shortID+"/edit?error=Failed+to+update", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (a *App) adminDeletePost(w http.ResponseWriter, r *http.Request) {
	_, _ = deletePost(a.DB, r.PathValue("id"))
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (a *App) adminTogglePublish(w http.ResponseWriter, r *http.Request) {
	_, _ = togglePostStatus(a.DB, r.PathValue("id"))
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (a *App) adminPages(w http.ResponseWriter, r *http.Request) {
	pages, err := getAllPages(a.DB)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	a.renderPage(w, "admin_pages.html", adminPagesPageData{Pages: pages})
}

func (a *App) adminNewPage(w http.ResponseWriter, r *http.Request) {
	a.renderPage(w, "admin_page_form.html", adminPageFormPageData{Error: r.URL.Query().Get("error")})
}

func (a *App) adminCreatePage(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/admin/pages/new?error=Bad+request", http.StatusSeeOther)
		return
	}
	attrs := parsePageAttributes(r.FormValue("attributes"))
	title := strings.TrimSpace(attrs.Title)
	slug := strings.TrimSpace(attrs.Slug)
	if title == "" || slug == "" {
		http.Redirect(w, r, "/admin/pages/new?error=Title+and+slug+are+required", http.StatusSeeOther)
		return
	}
	if isReservedPageSlug(slug) {
		http.Redirect(w, r, "/admin/pages/new?error=That+slug+is+reserved", http.StatusSeeOther)
		return
	}
	if _, err := createPage(a.DB, title, slug, r.FormValue("content"), attrs.IsPublished, 0); err != nil {
		a.Log.Error("create page", "err", err)
		http.Redirect(w, r, "/admin/pages/new?error=Failed+to+create+page", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin/pages", http.StatusSeeOther)
}

func (a *App) adminEditPage(w http.ResponseWriter, r *http.Request) {
	page, err := getPageByShortID(a.DB, r.PathValue("id"))
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	if page == nil {
		http.Error(w, "Page not found", http.StatusNotFound)
		return
	}
	a.renderPage(w, "admin_page_form.html", adminPageFormPageData{Page: page, Error: r.URL.Query().Get("error")})
}

func (a *App) adminUpdatePage(w http.ResponseWriter, r *http.Request) {
	shortID := r.PathValue("id")
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/admin/pages/"+shortID+"/edit?error=Bad+request", http.StatusSeeOther)
		return
	}
	attrs := parsePageAttributes(r.FormValue("attributes"))
	title := strings.TrimSpace(attrs.Title)
	slug := strings.TrimSpace(attrs.Slug)
	if title == "" || slug == "" {
		http.Redirect(w, r, "/admin/pages/"+shortID+"/edit?error=Title+and+slug+are+required", http.StatusSeeOther)
		return
	}
	if isReservedPageSlug(slug) {
		http.Redirect(w, r, "/admin/pages/"+shortID+"/edit?error=That+slug+is+reserved", http.StatusSeeOther)
		return
	}
	if _, err := updatePage(a.DB, shortID, title, slug, r.FormValue("content"), attrs.IsPublished, 0); err != nil {
		a.Log.Error("update page", "err", err)
		http.Redirect(w, r, "/admin/pages/"+shortID+"/edit?error=Failed+to+update", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin/pages", http.StatusSeeOther)
}

func (a *App) adminDeletePage(w http.ResponseWriter, r *http.Request) {
	_ = deletePage(a.DB, r.PathValue("id"))
	http.Redirect(w, r, "/admin/pages", http.StatusSeeOther)
}

func (a *App) adminGetSettings(w http.ResponseWriter, r *http.Request) {
	get := func(k string) string { v, _ := getSetting(a.DB, k); return v }
	defaultCSS, _ := appFS.ReadFile("static/styles.css")
	a.renderPage(w, "admin_settings.html", adminSettingsPageData{
		BlogTitle:       get("blog_title"),
		BlogDescription: get("blog_description"),
		IntroContent:    get("intro_content"),
		NavLinksRaw:     get("nav_links"),
		CustomCSS:       get("custom_css"),
		DefaultCSS:      string(defaultCSS),
		FaviconURL:      get("favicon_url"),
		OGImageURL:      get("og_image_url"),
		CustomHeader:    get("custom_header"),
		CustomFooter:    get("custom_footer"),
		DefaultLocation: get("default_location"),
		Success:         r.URL.Query().Get("success") == "true",
	})
}

func (a *App) adminPostSettings(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/admin/settings", http.StatusSeeOther)
		return
	}
	_ = setSetting(a.DB, "blog_title", strings.TrimSpace(r.FormValue("blog_title")))
	_ = setSetting(a.DB, "blog_description", strings.TrimSpace(r.FormValue("blog_description")))
	_ = setSetting(a.DB, "intro_content", r.FormValue("intro_content"))
	_ = setSetting(a.DB, "nav_links", r.FormValue("nav_links"))
	_ = setSetting(a.DB, "custom_css", r.FormValue("custom_css"))
	_ = setSetting(a.DB, "favicon_url", strings.TrimSpace(r.FormValue("favicon_url")))
	_ = setSetting(a.DB, "og_image_url", strings.TrimSpace(r.FormValue("og_image_url")))
	_ = setSetting(a.DB, "custom_header", r.FormValue("custom_header"))
	_ = setSetting(a.DB, "custom_footer", r.FormValue("custom_footer"))
	_ = setSetting(a.DB, "default_location", r.FormValue("default_location"))
	http.Redirect(w, r, "/admin/settings?success=true", http.StatusSeeOther)
}

func (a *App) adminFiles(w http.ResponseWriter, r *http.Request) {
	files, err := getAllFiles(a.DB)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	a.renderPage(w, "admin_files.html", adminFilesPageData{
		Files: files, SiteURL: a.SiteURL,
		Error:   r.URL.Query().Get("error"),
		Success: r.URL.Query().Get("success") == "true",
	})
}

func (a *App) adminUploadFile(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, bodyLimit)
	if err := r.ParseMultipartForm(uploadMaxBytes); err != nil {
		http.Redirect(w, r, "/admin/files?error=Failed+to+read+upload", http.StatusSeeOther)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Redirect(w, r, "/admin/files?error=No+file+provided", http.StatusSeeOther)
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		http.Redirect(w, r, "/admin/files?error=Failed+to+read+upload", http.StatusSeeOther)
		return
	}
	if int64(len(data)) > uploadMaxBytes {
		http.Redirect(w, r, "/admin/files?error=File+exceeds+10MB+limit", http.StatusSeeOther)
		return
	}
	originalName := "upload"
	contentType := "application/octet-stream"
	if header != nil {
		originalName = header.Filename
		if ct := header.Header.Get("Content-Type"); ct != "" {
			contentType = ct
		}
	}
	ext := ""
	if i := strings.LastIndex(originalName, "."); i > 0 && i < len(originalName)-1 {
		ext = originalName[i+1:]
	}
	id, _ := auth.GenerateShortID(10)
	stored := id
	if ext != "" {
		stored = id + "." + ext
	}
	backend := a.Storage
	if backend == nil {
		http.Redirect(w, r, "/admin/files?error=Storage+not+configured", http.StatusSeeOther)
		return
	}
	if err := backend.Put(r.Context(), stored, contentType, data); err != nil {
		a.Log.Error("save file", "backend", backend.Name(), "err", err)
		http.Redirect(w, r, "/admin/files?error=Failed+to+save+file", http.StatusSeeOther)
		return
	}
	if _, err := createFile(a.DB, stored, originalName, contentType, int64(len(data)), backend.Name()); err != nil {
		a.Log.Error("record file", "err", err)
		_ = backend.Delete(r.Context(), stored)
		http.Redirect(w, r, "/admin/files?error=Failed+to+record+file", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin/files?success=true", http.StatusSeeOther)
}

func (a *App) adminDeleteFile(w http.ResponseWriter, r *http.Request) {
	file, err := deleteFile(a.DB, r.PathValue("id"))
	if err != nil || file == nil {
		http.Redirect(w, r, "/admin/files", http.StatusSeeOther)
		return
	}
	if file.StorageBackend == "r2" && a.Storage != nil && a.Storage.Name() == "r2" {
		_ = a.Storage.Delete(r.Context(), file.Filename)
	} else {
		_ = removeFile(joinPath(a.UploadsDir, file.Filename))
	}
	http.Redirect(w, r, "/admin/files", http.StatusSeeOther)
}

func (a *App) adminDownloadPosts(w http.ResponseWriter, r *http.Request) {
	posts, err := getAllPosts(a.DB)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for i := range posts {
		f, err := zw.Create(posts[i].Slug + ".md")
		if err != nil {
			continue
		}
		_, _ = f.Write([]byte(postToMarkdown(&posts[i])))
	}
	_ = zw.Close()
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="posts.zip"`)
	_, _ = w.Write(buf.Bytes())
}

func (a *App) adminDownloadUploads(w http.ResponseWriter, r *http.Request) {
	files, err := getAllFiles(a.DB)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	header := &zip.FileHeader{Method: zip.Store}
	_ = header
	seen := map[string]bool{}
	for _, file := range files {
		data, err := readFileImpl(joinPath(a.UploadsDir, file.Filename))
		if err != nil {
			continue
		}
		name := file.OriginalName
		if seen[name] {
			name = file.ShortID + "_" + file.OriginalName
		}
		seen[file.OriginalName] = true
		w2, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Store})
		if err != nil {
			continue
		}
		_, _ = w2.Write(data)
	}
	_ = zw.Close()
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="uploads.zip"`)
	_, _ = w.Write(buf.Bytes())
}

func (a *App) adminImportForm(w http.ResponseWriter, r *http.Request) {
	data := adminImportPageData{Error: r.URL.Query().Get("error")}
	if v := r.URL.Query().Get("imported"); v != "" {
		var n int
		_, _ = fmt.Sscanf(v, "%d", &n)
		data.Imported = &n
	}
	if v := r.URL.Query().Get("skipped"); v != "" {
		var n int
		_, _ = fmt.Sscanf(v, "%d", &n)
		data.Skipped = &n
	}
	a.renderPage(w, "admin_import.html", data)
}

func (a *App) adminImportPosts(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, bodyLimit)
	if err := r.ParseMultipartForm(importMaxBytes); err != nil {
		http.Redirect(w, r, "/admin/import?error=Failed+to+read+upload", http.StatusSeeOther)
		return
	}
	file, _, err := r.FormFile("zip")
	if err != nil {
		http.Redirect(w, r, "/admin/import?error=No+zip+provided", http.StatusSeeOther)
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		http.Redirect(w, r, "/admin/import?error=Failed+to+read+upload", http.StatusSeeOther)
		return
	}
	if int64(len(data)) > importMaxBytes {
		http.Redirect(w, r, "/admin/import?error=Zip+exceeds+50MB+limit", http.StatusSeeOther)
		return
	}
	imported, skipped, err := a.processImportZip(data)
	if err != nil {
		a.Log.Error("import zip", "err", err)
		http.Redirect(w, r, "/admin/import?error=Invalid+zip+archive", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/admin/import?imported=%d&skipped=%d", imported, skipped), http.StatusSeeOther)
}

func (a *App) processImportZip(data []byte) (int, int, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return 0, 0, err
	}
	imported, skipped := 0, 0
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name := f.Name
		if strings.HasPrefix(name, "__MACOSX/") {
			continue
		}
		base := name
		if i := strings.LastIndex(name, "/"); i >= 0 {
			base = name[i+1:]
		}
		if base == "" || strings.HasPrefix(base, ".") {
			continue
		}
		low := strings.ToLower(base)
		if !strings.HasSuffix(low, ".md") && !strings.HasSuffix(low, ".markdown") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		raw, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}
		if a.importOne(base, string(raw), &imported, &skipped) {
			continue
		}
		skipped++
	}
	return imported, skipped, nil
}

func (a *App) importOne(basename, raw string, imported, skipped *int) bool {
	fm, body := splitFrontmatter(raw)
	attrs := parseAttributes(fm)
	title := strings.TrimSpace(attrs.Title)
	if title == "" {
		title = titleFromFilename(basename)
	}
	slug := deriveSlugWith(a, title, strings.TrimSpace(attrs.Slug))
	if slug == "" {
		return false
	}
	if existing, _ := getPostBySlug(a.DB, slug); existing != nil {
		*skipped++
		return true
	}
	status := "draft"
	if strings.EqualFold(strings.TrimSpace(attrs.Status), "published") {
		status = "published"
	}
	lang := "en"
	if l := strings.TrimSpace(attrs.Lang); l != "" {
		lang = l
	}
	pub := strings.TrimSpace(attrs.PublishedDate)
	if pub == "" {
		pub = nowDatetime()
	}
	in := PostInput{
		Title: optStr(title), Slug: slug, Content: body, Status: status,
		Alias:           optStr(attrs.Alias),
		PublishedDate:   &pub,
		MetaDescription: optStr(attrs.MetaDescription),
		MetaImage:       optStr(attrs.MetaImage),
		Lang:            lang, Tags: optStr(attrs.Tags),
	}
	if _, err := createPost(a.DB, in); err != nil {
		a.Log.Warn("import insert failed", "slug", slug, "err", err)
		return false
	}
	*imported++
	return true
}

var _ = url.QueryEscape
