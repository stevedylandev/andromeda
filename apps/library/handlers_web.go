package main

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/stevedylandev/andromeda/pkg/auth"
	"github.com/stevedylandev/andromeda/pkg/web"
)

var sectionDefs = []struct {
	Slug   string
	Status string
}{
	{"reading", "reading"},
	{"read", "read"},
	{"want", "want"},
}

func bookToView(b Book) bookView {
	v := bookView{ID: b.ID, Title: b.Title, Authors: b.Authors}
	if b.CoverURL != nil {
		v.CoverURL = *b.CoverURL
	}
	if b.Notes != nil {
		v.Notes = *b.Notes
	}
	if b.ISBN != nil {
		v.ISBN = *b.ISBN
	}
	return v
}

func quotesToViews(qs []Quote) []quoteView {
	out := make([]quoteView, 0, len(qs))
	for _, q := range qs {
		v := quoteView{ID: q.ID, BookID: q.BookID, Quote: q.Quote}
		if q.Page != nil {
			v.Page = *q.Page
		}
		out = append(out, v)
	}
	return out
}

func quoteToRow(q QuoteWithBook) adminQuoteRow {
	r := adminQuoteRow{ID: q.ID, BookID: q.BookID, Quote: q.Quote, BookTitle: q.BookTitle, BookAuthors: q.BookAuthors}
	if q.Page != nil {
		r.Page = *q.Page
	}
	return r
}

func quoteRow(q Quote) adminQuoteRow {
	r := adminQuoteRow{ID: q.ID, BookID: q.BookID, Quote: q.Quote}
	if q.Page != nil {
		r.Page = *q.Page
	}
	return r
}

// adminBackTo resolves the redirect target for admin quote form submissions
// via the hidden "from" field, falling back when absent or not an admin path.
func adminBackTo(r *http.Request, fallback string) string {
	from := r.FormValue("from")
	if strings.HasPrefix(from, "/admin/") {
		return from
	}
	return fallback
}

func bookToRow(b Book) adminBookRow {
	r := adminBookRow{ID: b.ID, Title: b.Title, Authors: b.Authors, Status: b.Status}
	if b.ISBN != nil {
		r.ISBN = *b.ISBN
	}
	if b.CoverURL != nil {
		r.CoverURL = *b.CoverURL
	}
	if b.Notes != nil {
		r.Notes = *b.Notes
	}
	return r
}

func (a *App) indexHandler(w http.ResponseWriter, r *http.Request) {
	all, _ := listBooks(a.DB, "")
	labels, _ := getCategoryLabels(a.DB)

	makeSection := func(status string) sectionView {
		sec := sectionView{Label: labelFor(labels, status)}
		for _, b := range all {
			if b.Status == status {
				sec.Books = append(sec.Books, bookToView(b))
			}
		}
		return sec
	}

	data := indexPageData{BaseURL: a.BaseURL, NavMode: a.DisplayMode == DisplayModeNav}
	if data.NavMode {
		selected := sectionDefs[0].Slug
		if q := r.URL.Query().Get("category"); q != "" {
			for _, s := range sectionDefs {
				if s.Slug == q {
					selected = q
					break
				}
			}
		}
		for _, s := range sectionDefs {
			data.NavCategories = append(data.NavCategories, navCategory{Slug: s.Slug, Label: labelFor(labels, s.Status), Active: s.Slug == selected})
		}
		for _, s := range sectionDefs {
			if s.Slug == selected {
				data.Sections = []sectionView{makeSection(s.Status)}
				break
			}
		}
	} else {
		for _, s := range sectionDefs {
			sec := makeSection(s.Status)
			if len(sec.Books) > 0 {
				data.Sections = append(data.Sections, sec)
			}
		}
	}
	web.Render(a.Templates, w, "index.html", data, a.Log)
}

func (a *App) loginGetHandler(w http.ResponseWriter, r *http.Request) {
	web.Render(a.Templates, w, "login.html", loginPageData{Error: r.URL.Query().Get("error")}, a.Log)
}

func (a *App) loginPostHandler(w http.ResponseWriter, r *http.Request) {
	if a.AdminPassword == "" {
		web.RedirectWithError(w, r, "/admin/login", "No admin password configured")
		return
	}
	if err := r.ParseForm(); err != nil {
		web.RedirectWithError(w, r, "/admin/login", "Bad request")
		return
	}
	if !auth.VerifyPassword(r.FormValue("password"), a.AdminPassword) {
		web.RedirectWithError(w, r, "/admin/login", "Invalid password")
		return
	}
	token, err := a.Sessions.Create()
	if err != nil {
		a.Log.Error("create session failed", "err", err)
		web.RedirectWithError(w, r, "/admin/login", "Session error")
		return
	}
	a.Sessions.PruneExpired()
	http.SetCookie(w, a.Sessions.SessionCookie(token))
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (a *App) logoutHandler(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(a.Sessions.CookieName); err == nil && c.Value != "" {
		a.Sessions.Delete(c.Value)
	}
	http.SetCookie(w, a.Sessions.ClearCookie())
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

func (a *App) adminHandler(w http.ResponseWriter, r *http.Request) {
	all, _ := listBooks(a.DB, "")
	labels, _ := getCategoryLabels(a.DB)
	rows := make([]adminBookRow, 0, len(all))
	for _, b := range all {
		rows = append(rows, bookToRow(b))
	}

	libraryQuery := r.URL.Query().Get("q")
	searched := strings.TrimSpace(libraryQuery) != ""
	var results []adminBookRow
	if searched {
		found, _ := searchBooks(a.DB, libraryQuery)
		results = make([]adminBookRow, 0, len(found))
		for _, b := range found {
			results = append(results, bookToRow(b))
		}
	}

	web.Render(a.Templates, w, "admin.html", adminPageData{
		Success:         r.URL.Query().Get("success"),
		Error:           r.URL.Query().Get("error"),
		Books:           rows,
		Labels:          labels,
		LibraryQuery:    libraryQuery,
		LibraryResults:  results,
		LibrarySearched: searched,
	}, a.Log)
}

func (a *App) adminUpdateLabels(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		web.RedirectWithError(w, r, "/admin", "Bad request")
		return
	}
	reading := strings.TrimSpace(r.FormValue("reading"))
	read := strings.TrimSpace(r.FormValue("read"))
	want := strings.TrimSpace(r.FormValue("want"))
	if reading == "" || read == "" || want == "" {
		web.RedirectWithError(w, r, "/admin", "Labels cannot be empty")
		return
	}
	if err := setSetting(a.DB, "category_label.reading", reading); err != nil {
		web.RedirectWithError(w, r, "/admin", "Failed to save labels")
		return
	}
	_ = setSetting(a.DB, "category_label.read", read)
	_ = setSetting(a.DB, "category_label.want", want)
	web.RedirectWithSuccess(w, r, "/admin", "Labels updated")
}

func (a *App) adminSearch(w http.ResponseWriter, r *http.Request) {
	hits, err := openLibrarySearch(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		a.Log.Warn("Open Library search failed", "err", err)
		web.WriteError(w, http.StatusBadGateway, err.Error())
		return
	}
	if hits == nil {
		hits = []SearchHit{}
	}
	web.WriteJSON(w, http.StatusOK, hits)
}

func (a *App) adminAddBook(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		web.RedirectWithError(w, r, "/admin", "Bad request")
		return
	}
	status := r.FormValue("status")
	if !validStatus(status) {
		web.RedirectWithError(w, r, "/admin", "Invalid status")
		return
	}
	b := NewBook{Title: r.FormValue("title"), Authors: r.FormValue("author_name"), Status: status}
	if v := strings.TrimSpace(r.FormValue("key")); v != "" {
		b.OLKey = &v
	}
	if v := strings.TrimSpace(r.FormValue("isbn")); v != "" {
		b.ISBN = &v
	}
	if v := strings.TrimSpace(r.FormValue("cover_url")); v != "" {
		b.CoverURL = &v
	}
	if _, err := insertBook(a.DB, b); err != nil {
		a.Log.Error("insert book", "err", err)
		web.RedirectWithError(w, r, "/admin", "Failed to add book")
		return
	}
	web.RedirectWithSuccess(w, r, "/admin", "Book added")
}

func (a *App) adminUpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		web.RedirectWithError(w, r, "/admin", "Bad id")
		return
	}
	if err := r.ParseForm(); err != nil {
		web.RedirectWithError(w, r, "/admin", "Bad request")
		return
	}
	status := r.FormValue("status")
	if !validStatus(status) {
		web.RedirectWithError(w, r, "/admin", "Invalid status")
		return
	}
	_ = updateBookStatus(a.DB, id, status)
	web.RedirectWithSuccess(w, r, "/admin", "Status updated")
}

func (a *App) adminUpdateNotes(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		web.RedirectWithError(w, r, "/admin", "Bad id")
		return
	}
	if err := r.ParseForm(); err != nil {
		web.RedirectWithError(w, r, "/admin", "Bad request")
		return
	}
	notes := strings.TrimSpace(r.FormValue("notes"))
	var n *string
	if notes != "" {
		n = &notes
	}
	_ = updateBookNotes(a.DB, id, n)
	web.RedirectWithSuccess(w, r, "/admin", "Notes saved")
}

func (a *App) adminDeleteBook(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err == nil {
		_ = deleteBook(a.DB, id)
	}
	web.RedirectWithSuccess(w, r, "/admin", "Book removed")
}

func (a *App) bookPageHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	b, err := getBook(a.DB, id)
	if err != nil {
		a.Log.Error("get book", "err", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	if b == nil {
		http.NotFound(w, r)
		return
	}
	quotes, err := listQuotesForBook(a.DB, id, true)
	if err != nil {
		a.Log.Error("list quotes for book", "err", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	web.Render(a.Templates, w, "book.html", bookPageData{
		BaseURL: a.BaseURL,
		Book:    bookToView(*b),
		Quotes:  quotesToViews(quotes),
	}, a.Log)
}

func (a *App) quotesHandler(w http.ResponseWriter, r *http.Request) {
	all, err := listQuotes(a.DB, true)
	if err != nil {
		a.Log.Error("list quotes", "err", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	data := quotesPageData{BaseURL: a.BaseURL}
	for _, qb := range all {
		var g *quoteGroup
		if len(data.Groups) > 0 {
			g = &data.Groups[len(data.Groups)-1]
		}
		if g == nil || g.BookID != qb.BookID {
			data.Groups = append(data.Groups, quoteGroup{BookID: qb.BookID, BookTitle: qb.BookTitle, BookAuthors: qb.BookAuthors})
			g = &data.Groups[len(data.Groups)-1]
		}
		qv := quoteView{ID: qb.ID, BookID: qb.BookID, Quote: qb.Quote}
		if qb.Page != nil {
			qv.Page = *qb.Page
		}
		g.Quotes = append(g.Quotes, qv)
	}
	web.Render(a.Templates, w, "quotes.html", data, a.Log)
}

func (a *App) quotePageHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	q, err := getQuote(a.DB, id)
	if err != nil {
		a.Log.Error("get quote", "err", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	if q == nil || !q.Public {
		http.NotFound(w, r)
		return
	}
	b, err := getBook(a.DB, q.BookID)
	if err != nil {
		a.Log.Error("get book", "err", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	if b == nil {
		http.NotFound(w, r)
		return
	}
	data := quotePageData{BaseURL: a.BaseURL, Book: bookToView(*b)}
	qv := quoteView{ID: q.ID, BookID: q.BookID, Quote: q.Quote}
	if q.Page != nil {
		qv.Page = *q.Page
	}
	data.Quote = qv
	web.Render(a.Templates, w, "quote.html", data, a.Log)
}

func (a *App) adminEditBookHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		web.RedirectWithError(w, r, "/admin", "Bad id")
		return
	}
	b, err := getBook(a.DB, id)
	if err != nil {
		a.Log.Error("get book", "err", err)
		web.RedirectWithError(w, r, "/admin", "Failed to load book")
		return
	}
	if b == nil {
		web.RedirectWithError(w, r, "/admin", "Book not found")
		return
	}
	quotes, err := listQuotesForBook(a.DB, id, false)
	if err != nil {
		a.Log.Error("list quotes for book", "err", err)
		web.RedirectWithError(w, r, "/admin", "Failed to load quotes")
		return
	}
	rows := make([]adminQuoteRow, 0, len(quotes))
	for _, q := range quotes {
		rows = append(rows, quoteRow(q))
	}
	web.Render(a.Templates, w, "admin_book.html", bookWithQuotesPageData{
		BaseURL: a.BaseURL,
		Success: r.URL.Query().Get("success"),
		Error:   r.URL.Query().Get("error"),
		Book:    bookToRow(*b),
		Quotes:  rows,
	}, a.Log)
}

func (a *App) adminUpdateBook(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	back := "/admin/books/" + strconv.FormatInt(id, 10)
	if err != nil {
		web.RedirectWithError(w, r, "/admin", "Bad id")
		return
	}
	if err := r.ParseForm(); err != nil {
		web.RedirectWithError(w, r, back, "Bad request")
		return
	}
	title := strings.TrimSpace(r.FormValue("title"))
	authors := strings.TrimSpace(r.FormValue("authors"))
	if title == "" || authors == "" {
		web.RedirectWithError(w, r, back, "Title and authors are required")
		return
	}
	if err := updateBookDetails(a.DB, id, title, authors, r.FormValue("isbn"), r.FormValue("cover_url"), r.FormValue("notes")); err != nil {
		a.Log.Error("update book details", "err", err)
		web.RedirectWithError(w, r, back, "Failed to save book")
		return
	}
	web.RedirectWithSuccess(w, r, back, "Book updated")
}

func (a *App) adminQuotesHandler(w http.ResponseWriter, r *http.Request) {
	all, err := listQuotes(a.DB, false)
	if err != nil {
		a.Log.Error("list quotes", "err", err)
		web.RedirectWithError(w, r, "/admin", "Failed to load quotes")
		return
	}
	rows := make([]adminQuoteRow, 0, len(all))
	for _, q := range all {
		rows = append(rows, quoteToRow(q))
	}
	books, err := listBooks(a.DB, "")
	if err != nil {
		a.Log.Error("list books", "err", err)
		web.RedirectWithError(w, r, "/admin", "Failed to load books")
		return
	}
	sort.Slice(books, func(i, j int) bool {
		return strings.ToLower(books[i].Title) < strings.ToLower(books[j].Title)
	})
	bookRows := make([]adminBookRow, 0, len(books))
	for _, b := range books {
		bookRows = append(bookRows, bookToRow(b))
	}
	web.Render(a.Templates, w, "admin_quotes.html", adminQuotesPageData{
		BaseURL: a.BaseURL,
		Success: r.URL.Query().Get("success"),
		Error:   r.URL.Query().Get("error"),
		Quotes:  rows,
		Books:   bookRows,
	}, a.Log)
}

// validQuoteForm validates and extracts quote fields shared by add and edit.
func validQuoteForm(r *http.Request) (NewQuote, string) {
	bookID, err := strconv.ParseInt(r.FormValue("book_id"), 10, 64)
	if err != nil || bookID <= 0 {
		return NewQuote{}, "Bad book"
	}
	quote := strings.TrimSpace(r.FormValue("quote"))
	if quote == "" {
		return NewQuote{}, "Quote is required"
	}
	page := strings.TrimSpace(r.FormValue("page"))
	nq := NewQuote{BookID: bookID, Quote: quote}
	if page != "" {
		nq.Page = &page
	}
	return nq, ""
}

func (a *App) adminAddQuote(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		web.RedirectWithError(w, r, "/admin/quotes", "Bad request")
		return
	}
	back := adminBackTo(r, "/admin/quotes")
	nq, msg := validQuoteForm(r)
	if msg != "" {
		web.RedirectWithError(w, r, back, msg)
		return
	}
	b, err := getBook(a.DB, nq.BookID)
	if err != nil {
		a.Log.Error("get book", "err", err)
		web.RedirectWithError(w, r, back, "Failed to load book")
		return
	}
	if b == nil {
		web.RedirectWithError(w, r, back, "Book not found")
		return
	}
	if _, err := insertQuote(a.DB, nq); err != nil {
		a.Log.Error("insert quote", "err", err)
		web.RedirectWithError(w, r, back, "Failed to add quote")
		return
	}
	web.RedirectWithSuccess(w, r, back, "Quote added")
}

func (a *App) adminUpdateQuote(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		web.RedirectWithError(w, r, "/admin/quotes", "Bad id")
		return
	}
	if q, gerr := getQuote(a.DB, id); gerr != nil {
		a.Log.Error("get quote", "err", gerr)
		web.RedirectWithError(w, r, "/admin/quotes", "Failed to load quote")
		return
	} else if q == nil {
		web.RedirectWithError(w, r, "/admin/quotes", "Quote not found")
		return
	}
	if err := r.ParseForm(); err != nil {
		web.RedirectWithError(w, r, "/admin/quotes", "Bad request")
		return
	}
	back := adminBackTo(r, "/admin/quotes")
	nq, msg := validQuoteForm(r)
	if msg != "" {
		web.RedirectWithError(w, r, back, msg)
		return
	}
	b, err := getBook(a.DB, nq.BookID)
	if err != nil {
		a.Log.Error("get book", "err", err)
		web.RedirectWithError(w, r, back, "Failed to load book")
		return
	}
	if b == nil {
		web.RedirectWithError(w, r, back, "Book not found")
		return
	}
	if err := updateQuote(a.DB, id, nq); err != nil {
		a.Log.Error("update quote", "err", err)
		web.RedirectWithError(w, r, back, "Failed to save quote")
		return
	}
	web.RedirectWithSuccess(w, r, back, "Quote updated")
}

func (a *App) adminDeleteQuote(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		web.RedirectWithError(w, r, "/admin/quotes", "Bad request")
		return
	}
	back := adminBackTo(r, "/admin/quotes")
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err == nil {
		_ = deleteQuote(a.DB, id)
	}
	web.RedirectWithSuccess(w, r, back, "Quote deleted")
}
