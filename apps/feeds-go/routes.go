package main

import "net/http"

func (a *App) routes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", a.indexHandler)
	mux.HandleFunc("GET /feeds", a.feedsExportHandler)
	mux.HandleFunc("GET /feed.xml", a.atomFeedHandler)
	mux.HandleFunc("GET /static/", a.embeddedHandler("static"))
	mux.HandleFunc("GET /assets/", a.embeddedHandler("assets"))

	mux.HandleFunc("GET /admin/login", a.loginGetHandler)
	mux.HandleFunc("POST /admin/login", a.loginPostHandler)
	mux.HandleFunc("GET /admin/logout", a.logoutHandler)
	mux.HandleFunc("GET /admin", a.requireSession(a.adminHandler))
	mux.HandleFunc("POST /admin/add-feed", a.requireSession(a.addFeedHandler))
	mux.HandleFunc("POST /admin/feeds/{id}/delete", a.requireSession(a.deleteFeedHandler))
	mux.HandleFunc("POST /admin/feeds/{id}/category", a.requireSession(a.updateSubCategoryHandler))
	mux.HandleFunc("POST /admin/categories", a.requireSession(a.addCategoryHandler))
	mux.HandleFunc("POST /admin/categories/{id}/delete", a.requireSession(a.deleteCategoryHandler))
	mux.HandleFunc("POST /admin/import-opml", a.requireSession(a.importOPMLHandler))
	mux.HandleFunc("POST /admin/settings", a.requireSession(a.updateSettingsFormHandler))
	mux.HandleFunc("POST /admin/discover-feeds", a.requireSession(a.discoverFeedsHandler))

	mux.HandleFunc("GET /api/items", a.withCORS(a.listItemsAPI))
	mux.HandleFunc("POST /api/items/{id}/read", a.withCORS(a.requireAPIAuth(a.markItemReadAPI(true))))
	mux.HandleFunc("POST /api/items/{id}/unread", a.withCORS(a.requireAPIAuth(a.markItemReadAPI(false))))
	mux.HandleFunc("GET /api/subscriptions", a.withCORS(a.listSubscriptionsAPI))
	mux.HandleFunc("POST /api/subscriptions", a.withCORS(a.requireAPIAuth(a.createSubscriptionAPI)))
	mux.HandleFunc("PATCH /api/subscriptions/{id}", a.withCORS(a.requireAPIAuth(a.updateSubscriptionAPI)))
	mux.HandleFunc("DELETE /api/subscriptions/{id}", a.withCORS(a.requireAPIAuth(a.deleteSubscriptionAPI)))
	mux.HandleFunc("GET /api/categories", a.withCORS(a.listCategoriesAPI))
	mux.HandleFunc("POST /api/categories", a.withCORS(a.requireAPIAuth(a.createCategoryAPI)))
	mux.HandleFunc("DELETE /api/categories/{id}", a.withCORS(a.requireAPIAuth(a.deleteCategoryAPI)))
	mux.HandleFunc("POST /api/import/opml", a.withCORS(a.requireAPIAuth(a.importOPMLAPI)))
	mux.HandleFunc("GET /api/settings", a.withCORS(a.getSettingsAPI))
	mux.HandleFunc("PUT /api/settings", a.withCORS(a.requireAPIAuth(a.updateSettingsAPI)))
	mux.HandleFunc("POST /api/discover", a.withCORS(a.requireAPIAuth(a.discoverAPI)))

	return mux
}
