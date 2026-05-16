package main

import (
	"net/http"
	"strings"

	"github.com/stevedylandev/andromeda/crates-go/darkmatter"
	"github.com/stevedylandev/andromeda/crates-go/web"
)

func (a *App) routes() *http.ServeMux {
	mux := http.NewServeMux()

	requireSession := func(next http.HandlerFunc) http.HandlerFunc {
		return a.Sessions.RequireSession("/admin/login", next)
	}
	cors := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET")
			w.Header().Set("Access-Control-Allow-Headers", "*")
			next(w, r)
		}
	}

	// Public
	mux.HandleFunc("GET /{$}", a.publicIndex)
	mux.HandleFunc("GET /posts", a.publicPostsList)
	mux.HandleFunc("GET /posts/{slug}", a.publicPost)
	mux.HandleFunc("GET /custom-styles.css", a.customCSS)
	mux.HandleFunc("GET /feed.xml", a.rssFeed)
	mux.HandleFunc("GET /files/{filename}", a.serveUploadedFile)
	mux.HandleFunc("GET /static/", web.EmbeddedHandler(appFS, "static"))
	darkmatter.Mount(mux, "/assets")

	// API
	mux.HandleFunc("GET /api/posts", cors(a.apiListPosts))
	mux.HandleFunc("GET /api/posts/{slug}", cors(a.apiGetPost))

	// Admin auth
	mux.HandleFunc("GET /admin/login", a.loginGet)
	mux.HandleFunc("POST /admin/login", a.loginPost)
	mux.HandleFunc("GET /admin/logout", a.logout)

	// Admin posts
	mux.HandleFunc("GET /admin", requireSession(a.adminIndex))
	mux.HandleFunc("GET /admin/posts/new", requireSession(a.adminNewPost))
	mux.HandleFunc("POST /admin/posts", requireSession(a.adminCreatePost))
	mux.HandleFunc("GET /admin/posts/{id}/edit", requireSession(a.adminEditPost))
	mux.HandleFunc("POST /admin/posts/{id}", requireSession(a.adminUpdatePost))
	mux.HandleFunc("POST /admin/posts/{id}/delete", requireSession(a.adminDeletePost))
	mux.HandleFunc("POST /admin/posts/{id}/publish", requireSession(a.adminTogglePublish))

	// Admin pages
	mux.HandleFunc("GET /admin/pages", requireSession(a.adminPages))
	mux.HandleFunc("GET /admin/pages/new", requireSession(a.adminNewPage))
	mux.HandleFunc("POST /admin/pages/create", requireSession(a.adminCreatePage))
	mux.HandleFunc("GET /admin/pages/{id}/edit", requireSession(a.adminEditPage))
	mux.HandleFunc("POST /admin/pages/{id}", requireSession(a.adminUpdatePage))
	mux.HandleFunc("POST /admin/pages/{id}/delete", requireSession(a.adminDeletePage))

	// Settings
	mux.HandleFunc("GET /admin/settings", requireSession(a.adminGetSettings))
	mux.HandleFunc("POST /admin/settings", requireSession(a.adminPostSettings))

	// Downloads
	mux.HandleFunc("GET /admin/downloads/posts", requireSession(a.adminDownloadPosts))
	mux.HandleFunc("GET /admin/downloads/uploads", requireSession(a.adminDownloadUploads))

	// Import
	mux.HandleFunc("GET /admin/import", requireSession(a.adminImportForm))
	mux.HandleFunc("POST /admin/import", requireSession(a.adminImportPosts))

	// Files
	mux.HandleFunc("GET /admin/files", requireSession(a.adminFiles))
	mux.HandleFunc("POST /admin/files/upload", requireSession(a.adminUploadFile))
	mux.HandleFunc("POST /admin/files/{id}/delete", requireSession(a.adminDeleteFile))

	// Fallback: /{slug} → page or alias redirect
	mux.HandleFunc("GET /{slug}", a.publicPage)

	_ = strings.Trim
	return mux
}
