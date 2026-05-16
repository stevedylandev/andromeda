package main

import (
	"net/http"

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
			next(w, r)
		}
	}

	// Public
	mux.HandleFunc("GET /", a.indexHandler)
	mux.HandleFunc("GET /feed.xml", a.rssFeed)
	mux.HandleFunc("GET /wines/{short_id}", a.wineDetail)
	mux.HandleFunc("GET /wines/{short_id}/image", a.wineImage)
	mux.HandleFunc("GET /wishlist", a.wishlistHandler)

	// API
	mux.HandleFunc("GET /api/wines", cors(a.apiListWines))
	mux.HandleFunc("GET /api/wines/{short_id}", cors(a.apiGetWine))
	mux.HandleFunc("GET /api/wines/{short_id}/pentagon.svg", cors(a.apiPentagonSVG))
	mux.HandleFunc("GET /api/wines/{short_id}/bars.svg", cors(a.apiBarsSVG))

	// Admin auth
	mux.HandleFunc("GET /admin/login", a.loginGet)
	mux.HandleFunc("POST /admin/login", a.loginPost)
	mux.HandleFunc("GET /admin/logout", a.logout)

	// Admin protected
	mux.HandleFunc("GET /admin", requireSession(a.adminIndex))
	mux.HandleFunc("GET /admin/new", requireSession(a.newWineGet))
	mux.HandleFunc("POST /admin/new", requireSession(a.newWinePost))
	mux.HandleFunc("GET /admin/edit/{short_id}", requireSession(a.editWineGet))
	mux.HandleFunc("POST /admin/edit/{short_id}", requireSession(a.editWinePost))
	mux.HandleFunc("POST /admin/delete/{short_id}", requireSession(a.deleteWinePost))
	mux.HandleFunc("GET /admin/wishlist/new", requireSession(a.newWishlistGet))
	mux.HandleFunc("POST /admin/wishlist/new", requireSession(a.newWishlistPost))
	mux.HandleFunc("GET /admin/wishlist/edit/{short_id}", requireSession(a.editWishlistGet))
	mux.HandleFunc("POST /admin/wishlist/edit/{short_id}", requireSession(a.editWishlistPost))
	mux.HandleFunc("POST /admin/wishlist/delete/{short_id}", requireSession(a.deleteWishlistPost))
	mux.HandleFunc("POST /admin/wishlist/promote/{short_id}", requireSession(a.promoteWinePost))
	mux.HandleFunc("POST /admin/analyze-image", requireSession(a.analyzeImage))

	// Static
	mux.HandleFunc("GET /static/", web.EmbeddedHandler(appFS, "static"))
	darkmatter.Mount(mux, "/assets")
	return mux
}
