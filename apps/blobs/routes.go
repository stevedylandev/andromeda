package main

import (
	"net/http"

	"github.com/stevedylandev/andromeda/pkg/darkmatter"
	"github.com/stevedylandev/andromeda/pkg/web"
)

func (a *App) routes() *http.ServeMux {
	mux := http.NewServeMux()

	requireSession := func(next http.HandlerFunc) http.HandlerFunc {
		return a.Sessions.RequireSession("/login", next)
	}

	mux.HandleFunc("GET /static/", web.EmbeddedHandler(appFS, "static"))
	darkmatter.Mount(mux, "/assets")

	mux.HandleFunc("GET /login", a.loginGet)
	mux.HandleFunc("POST /login", a.loginPost)
	mux.HandleFunc("POST /logout", a.logout)

	mux.HandleFunc("GET /{$}", requireSession(a.rootRedirect))
	mux.HandleFunc("GET /buckets", requireSession(a.listBuckets))

	mux.HandleFunc("GET /b/{bucket}", requireSession(a.bucketRoot))
	mux.HandleFunc("GET /b/{bucket}/browse/{prefix...}", requireSession(a.browse))
	mux.HandleFunc("GET /b/{bucket}/object/{key...}", requireSession(a.detail))
	mux.HandleFunc("GET /b/{bucket}/preview/{key...}", requireSession(a.preview))

	mux.HandleFunc("POST /b/{bucket}/upload", requireSession(a.upload))
	mux.HandleFunc("POST /b/{bucket}/replace", requireSession(a.replace))
	mux.HandleFunc("POST /b/{bucket}/delete", requireSession(a.deleteObject))
	mux.HandleFunc("POST /b/{bucket}/mkdir", requireSession(a.mkdir))

	return mux
}
