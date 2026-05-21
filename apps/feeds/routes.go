package main

import (
	"net/http"

	"github.com/stevedylandev/andromeda/crates-go/auth"
	"github.com/stevedylandev/andromeda/crates-go/darkmatter"
	"github.com/stevedylandev/andromeda/crates-go/web"
)

func (a *App) routes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", a.indexHandler)
	mux.HandleFunc("GET /feeds", a.feedsExportHandler)
	mux.HandleFunc("GET /feed.xml", a.atomFeedHandler)
	mux.HandleFunc("GET /static/", web.EmbeddedHandler(appFS, "static"))
	darkmatter.Mount(mux, "/assets")

	requireSession := func(next http.HandlerFunc) http.HandlerFunc {
		return a.Sessions.RequireSession("/admin/login", next)
	}
	requireAPIAuth := func(next http.HandlerFunc) http.HandlerFunc {
		return auth.RequireBearerOrSession(a.Sessions, a.APIKey, next)
	}

	mux.HandleFunc("GET /admin/login", a.loginGetHandler)
	mux.HandleFunc("POST /admin/login", a.loginPostHandler)
	mux.HandleFunc("GET /admin/logout", a.logoutHandler)
	mux.HandleFunc("GET /admin", requireSession(a.adminHandler))
	mux.HandleFunc("POST /admin/add-feed", requireSession(a.addFeedHandler))
	mux.HandleFunc("POST /admin/feeds/{id}/delete", requireSession(a.deleteFeedHandler))
	mux.HandleFunc("POST /admin/feeds/{id}/category", requireSession(a.updateSubCategoryHandler))
	mux.HandleFunc("POST /admin/categories", requireSession(a.addCategoryHandler))
	mux.HandleFunc("POST /admin/categories/{id}/delete", requireSession(a.deleteCategoryHandler))
	mux.HandleFunc("POST /admin/import-opml", requireSession(a.importOPMLHandler))
	mux.HandleFunc("POST /admin/settings", requireSession(a.updateSettingsFormHandler))
	mux.HandleFunc("POST /admin/discover-feeds", requireSession(a.discoverFeedsHandler))

	mux.HandleFunc("GET /api/items", a.withCORS(a.listItemsAPI))
	mux.HandleFunc("POST /api/items/{id}/read", a.withCORS(requireAPIAuth(a.markItemReadAPI(true))))
	mux.HandleFunc("POST /api/items/{id}/unread", a.withCORS(requireAPIAuth(a.markItemReadAPI(false))))
	mux.HandleFunc("GET /api/subscriptions", a.withCORS(a.listSubscriptionsAPI))
	mux.HandleFunc("POST /api/subscriptions", a.withCORS(requireAPIAuth(a.createSubscriptionAPI)))
	mux.HandleFunc("PATCH /api/subscriptions/{id}", a.withCORS(requireAPIAuth(a.updateSubscriptionAPI)))
	mux.HandleFunc("DELETE /api/subscriptions/{id}", a.withCORS(requireAPIAuth(a.deleteSubscriptionAPI)))
	mux.HandleFunc("GET /api/categories", a.withCORS(a.listCategoriesAPI))
	mux.HandleFunc("POST /api/categories", a.withCORS(requireAPIAuth(a.createCategoryAPI)))
	mux.HandleFunc("DELETE /api/categories/{id}", a.withCORS(requireAPIAuth(a.deleteCategoryAPI)))
	mux.HandleFunc("POST /api/import/opml", a.withCORS(requireAPIAuth(a.importOPMLAPI)))
	mux.HandleFunc("GET /api/settings", a.withCORS(a.getSettingsAPI))
	mux.HandleFunc("PUT /api/settings", a.withCORS(requireAPIAuth(a.updateSettingsAPI)))
	mux.HandleFunc("POST /api/discover", a.withCORS(requireAPIAuth(a.discoverAPI)))

	return mux
}
