// Package server hosts the sipp web server, API, and admin pages.
package server

import (
	"bytes"
	"database/sql"
	"embed"
	"encoding/json"
	"html/template"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/stevedylandev/andromeda/apps/sipp/internal/store"
	"github.com/stevedylandev/andromeda/pkg/auth"
	"github.com/stevedylandev/andromeda/pkg/config"
	"github.com/stevedylandev/andromeda/pkg/darkmatter"
	"github.com/stevedylandev/andromeda/pkg/web"
)

//go:embed templates/*.html static/*
var appFS embed.FS

var siteStyle = chroma.MustNewStyle("site", chroma.StyleEntries{
	chroma.Background:               "#c1c1c1 bg:#121113",
	chroma.Comment:                  "italic #444444",
	chroma.CommentHashbang:          "italic #444444",
	chroma.CommentMultiline:         "italic #444444",
	chroma.CommentSingle:            "italic #444444",
	chroma.CommentSpecial:           "italic #444444",
	chroma.CommentPreproc:           "italic #444444",
	chroma.Keyword:                  "#999999",
	chroma.KeywordConstant:          "#e78a52",
	chroma.KeywordDeclaration:       "#e78a52",
	chroma.KeywordNamespace:         "#e78a52",
	chroma.KeywordPseudo:            "#999999",
	chroma.KeywordReserved:          "#999999",
	chroma.KeywordType:              "#e78a52",
	chroma.LiteralString:            "#fbcb96",
	chroma.LiteralStringAffix:       "#fbcb96",
	chroma.LiteralStringBacktick:    "#fbcb96",
	chroma.LiteralStringChar:        "#fbcb96",
	chroma.LiteralStringDelimiter:   "#fbcb96",
	chroma.LiteralStringDoc:         "#fbcb96",
	chroma.LiteralStringDouble:      "#fbcb96",
	chroma.LiteralStringEscape:      "#c1c1c1",
	chroma.LiteralStringHeredoc:     "#fbcb96",
	chroma.LiteralStringInterpol:    "#fbcb96",
	chroma.LiteralStringOther:       "#fbcb96",
	chroma.LiteralStringRegex:       "#c1c1c1",
	chroma.LiteralStringSingle:      "#fbcb96",
	chroma.LiteralStringSymbol:      "#c1c1c1",
	chroma.LiteralNumber:            "#c1c1c1",
	chroma.LiteralNumberBin:         "#c1c1c1",
	chroma.LiteralNumberFloat:       "#c1c1c1",
	chroma.LiteralNumberHex:         "#c1c1c1",
	chroma.LiteralNumberInteger:     "#c1c1c1",
	chroma.LiteralNumberIntegerLong: "#c1c1c1",
	chroma.LiteralNumberOct:         "#c1c1c1",
	chroma.NameFunction:             "#aaabab",
	chroma.NameFunctionMagic:        "#aaabab",
	chroma.NameClass:                "#e78a52",
	chroma.NameNamespace:            "#e78a52",
	chroma.NameConstant:             "#e78a52",
	chroma.NameDecorator:            "#e78a52",
	chroma.NameBuiltin:              "#c1c1c1",
	chroma.NameBuiltinPseudo:        "#e78a52",
	chroma.NameAttribute:            "#c1c1c1",
	chroma.NameTag:                  "#5f8787",
	chroma.NameVariable:             "#5f8787",
	chroma.NameVariableClass:        "#5f8787",
	chroma.NameVariableGlobal:       "#5f8787",
	chroma.NameVariableInstance:     "#5f8787",
	chroma.NameOther:                "#c1c1c1",
	chroma.Operator:                 "#999999",
	chroma.OperatorWord:             "#999999",
	chroma.Punctuation:              "#999999",
	chroma.GenericDeleted:           "#e78a52",
	chroma.GenericInserted:          "#fbcb96",
	chroma.GenericHeading:           "bold #fbcb96",
	chroma.GenericSubheading:        "bold #fbcb96",
	chroma.GenericEmph:              "italic",
	chroma.GenericStrong:            "bold",
	chroma.Error:                    "#e78a52",
	chroma.LineNumbers:              "#444444",
	chroma.LineNumbersTable:         "#444444",
})

type Snippet = store.Snippet

type App struct {
	DB             *sql.DB
	Log            *slog.Logger
	Templates      *template.Template
	Sessions       *auth.Store
	APIKey         string
	BaseURL        string
	CookieSecure   bool
	AuthEndpoints  map[string]bool
	MaxContentSize int
}

var (
	createSnippet          = store.Create
	getSnippetByShortID    = store.GetByShortID
	getAllSnippets         = store.List
	deleteSnippetByShortID = store.DeleteByShortID
	updateSnippetByShortID = store.UpdateByShortID
)

func highlight(name, content string) string {
	ext := ""
	if i := strings.LastIndex(name, "."); i >= 0 && i < len(name)-1 {
		ext = strings.ToLower(name[i+1:])
	}
	switch ext {
	case "ts", "tsx", "jsx":
		ext = "js"
	}
	var lexer chroma.Lexer
	if ext != "" {
		lexer = lexers.MatchMimeType("text/" + ext)
		if lexer == nil {
			lexer = lexers.Get(ext)
		}
	}
	if lexer == nil {
		lexer = lexers.Analyse(content)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	style := siteStyle
	if style == nil {
		style = styles.Fallback
	}
	formatter := html.New(html.Standalone(false), html.WithClasses(false))
	iterator, err := lexer.Tokenise(nil, content)
	if err != nil {
		escaped := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(content)
		return "<pre>" + escaped + "</pre>"
	}
	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iterator); err != nil {
		escaped := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(content)
		return "<pre>" + escaped + "</pre>"
	}
	return buf.String()
}

type indexPageData struct{ BaseURL string }
type adminPageData struct {
	BaseURL  string
	Snippets []Snippet
}
type loginPageData struct {
	Error string
	Next  string
}
type snippetPageData struct {
	BaseURL            string
	Name               string
	Content            string
	HighlightedContent template.HTML
}

func (a *App) indexHandler(w http.ResponseWriter, r *http.Request) {
	web.Render(a.Templates, w, "index.html", indexPageData{BaseURL: a.BaseURL}, a.Log)
}

func (a *App) adminHandler(w http.ResponseWriter, r *http.Request) {
	snippets, err := getAllSnippets(a.DB)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	web.Render(a.Templates, w, "admin.html", adminPageData{BaseURL: a.BaseURL, Snippets: snippets}, a.Log)
}

func (a *App) loginGet(w http.ResponseWriter, r *http.Request) {
	web.Render(a.Templates, w, "login.html", loginPageData{Error: r.URL.Query().Get("error"), Next: r.URL.Query().Get("next")}, a.Log)
}

func (a *App) loginPost(w http.ResponseWriter, r *http.Request) {
	next := r.URL.Query().Get("next")
	if next == "" {
		next = "/admin"
	}
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/admin/login?error=Bad+request", http.StatusSeeOther)
		return
	}
	if a.APIKey == "" {
		http.Redirect(w, r, "/admin/login?error=No+API+key+configured", http.StatusSeeOther)
		return
	}
	if !auth.SecureEqual(r.FormValue("api_key"), a.APIKey) {
		http.Redirect(w, r, "/admin/login?error=Invalid+API+key&next="+url.QueryEscape(next), http.StatusSeeOther)
		return
	}
	token, err := a.Sessions.Create()
	if err != nil {
		http.Redirect(w, r, "/admin/login?error=Server+error", http.StatusSeeOther)
		return
	}
	a.Sessions.PruneExpired()
	http.SetCookie(w, a.Sessions.SessionCookie(token))
	target := "/admin"
	if strings.HasPrefix(next, "/") {
		target = next
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(a.Sessions.CookieName); err == nil && c.Value != "" {
		a.Sessions.Delete(c.Value)
	}
	http.SetCookie(w, a.Sessions.ClearCookie())
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

func (a *App) adminDeleteSnippet(w http.ResponseWriter, r *http.Request) {
	_, _ = deleteSnippetByShortID(a.DB, r.PathValue("short_id"))
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func isCLIUserAgent(r *http.Request) bool {
	ua := strings.ToLower(r.Header.Get("User-Agent"))
	return strings.HasPrefix(ua, "curl/") || strings.HasPrefix(ua, "wget/") || strings.HasPrefix(ua, "httpie/")
}

func (a *App) viewSnippet(w http.ResponseWriter, r *http.Request) {
	snippet, err := getSnippetByShortID(a.DB, r.PathValue("short_id"))
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if snippet == nil {
		http.Error(w, "Snippet not found", http.StatusNotFound)
		return
	}
	if isCLIUserAgent(r) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(snippet.Content))
		return
	}
	highlighted := highlight(snippet.Name, snippet.Content)
	web.Render(a.Templates, w, "snippet.html", snippetPageData{
		BaseURL:            a.BaseURL,
		Name:               snippet.Name,
		Content:            snippet.Content,
		HighlightedContent: template.HTML(highlighted),
	}, a.Log)
}

func (a *App) createSnippetForm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	content := r.FormValue("content")
	if len(content) > a.MaxContentSize {
		http.Error(w, "Content too large", http.StatusRequestEntityTooLarge)
		return
	}
	sn, err := createSnippet(a.DB, r.FormValue("name"), content)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/s/"+sn.ShortID, http.StatusSeeOther)
}

func (a *App) requireAPIKey(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if a.APIKey == "" {
			web.WriteError(w, http.StatusForbidden, "No API key configured on server")
			return
		}
		if auth.SecureEqual(r.Header.Get("x-api-key"), a.APIKey) {
			next(w, r)
			return
		}
		if a.Sessions.HasValid(r) {
			next(w, r)
			return
		}
		web.WriteError(w, http.StatusUnauthorized, "Invalid or missing API key")
	}
}

func (a *App) apiList(w http.ResponseWriter, r *http.Request) {
	snippets, err := getAllSnippets(a.DB)
	if err != nil {
		web.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	web.WriteJSON(w, http.StatusOK, snippets)
}

func (a *App) apiGet(w http.ResponseWriter, r *http.Request) {
	s, err := getSnippetByShortID(a.DB, r.PathValue("short_id"))
	if err != nil {
		web.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	if s == nil {
		web.WriteError(w, http.StatusNotFound, "Snippet not found")
		return
	}
	web.WriteJSON(w, http.StatusOK, s)
}

type apiCreateBody struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

func (a *App) apiCreate(w http.ResponseWriter, r *http.Request) {
	var body apiCreateBody
	if !web.DecodeJSON(w, r, &body) {
		return
	}
	if len(body.Content) > a.MaxContentSize {
		web.WriteError(w, http.StatusRequestEntityTooLarge, "Content too large. Maximum size is "+strconv.Itoa(a.MaxContentSize)+" bytes")
		return
	}
	s, err := createSnippet(a.DB, body.Name, body.Content)
	if err != nil {
		web.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	web.WriteJSON(w, http.StatusCreated, s)
}

func (a *App) apiUpdate(w http.ResponseWriter, r *http.Request) {
	var body apiCreateBody
	if !web.DecodeJSON(w, r, &body) {
		return
	}
	if len(body.Content) > a.MaxContentSize {
		web.WriteError(w, http.StatusRequestEntityTooLarge, "Content too large")
		return
	}
	s, err := updateSnippetByShortID(a.DB, r.PathValue("short_id"), body.Name, body.Content)
	if err != nil {
		web.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	if s == nil {
		web.WriteError(w, http.StatusNotFound, "Snippet not found")
		return
	}
	web.WriteJSON(w, http.StatusOK, s)
}

func (a *App) apiDelete(w http.ResponseWriter, r *http.Request) {
	ok, err := deleteSnippetByShortID(a.DB, r.PathValue("short_id"))
	if err != nil {
		web.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	if !ok {
		web.WriteError(w, http.StatusNotFound, "Snippet not found")
		return
	}
	web.WriteJSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (a *App) requiresAuth(name string) bool {
	return a.AuthEndpoints["all"] || a.AuthEndpoints[name]
}

func (a *App) wrapIfAuth(name string, h http.HandlerFunc) http.HandlerFunc {
	if a.requiresAuth(name) {
		return a.requireAPIKey(h)
	}
	return h
}

func parseAuthEndpoints(raw string) map[string]bool {
	out := map[string]bool{}
	if strings.EqualFold(strings.TrimSpace(raw), "none") {
		return out
	}
	if raw == "" {
		out["api_delete"] = true
		out["api_list"] = true
		out["api_update"] = true
		return out
	}
	for _, p := range strings.Split(raw, ",") {
		if v := strings.ToLower(strings.TrimSpace(p)); v != "" {
			out[v] = true
		}
	}
	return out
}

// Run starts the sipp web server.
func Run(host string, port int) error {
	config.LoadDotEnv(".env")
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	dbPath := config.Getenv("SIPP_DB_PATH", "sipp.sqlite")
	db, err := store.Open(dbPath)
	if err != nil {
		return err
	}

	apiKey := os.Getenv("SIPP_API_KEY")
	authEndpoints := parseAuthEndpoints(os.Getenv("SIPP_AUTH_ENDPOINTS"))
	maxSize := config.GetenvInt("SIPP_MAX_CONTENT_SIZE", 512000)
	baseURL := strings.TrimRight(config.Getenv("BASE_URL", "http://localhost:3000"), "/")
	cookieSecure := config.GetenvBool("SIPP_COOKIE_SECURE", false)

	if len(authEndpoints) > 0 && apiKey == "" {
		logger.Warn("SIPP_AUTH_ENDPOINTS set but SIPP_API_KEY not configured")
	}

	sessions := &auth.Store{DB: db, CookieName: "session", CookieSecure: cookieSecure}
	if err := sessions.EnsureSchema(); err != nil {
		return err
	}
	sessions.PruneExpired()

	tmpl := template.Must(template.ParseFS(appFS, "templates/*.html"))

	app := &App{
		DB: db, Log: logger, Templates: tmpl, Sessions: sessions,
		APIKey: apiKey, BaseURL: baseURL, CookieSecure: cookieSecure,
		AuthEndpoints: authEndpoints, MaxContentSize: maxSize,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", app.indexHandler)
	mux.HandleFunc("GET /admin", app.Sessions.RequireSession("/admin/login", app.adminHandler))
	mux.HandleFunc("GET /admin/login", app.loginGet)
	mux.HandleFunc("POST /admin/login", app.loginPost)
	mux.HandleFunc("POST /admin/logout", app.logout)
	mux.HandleFunc("POST /admin/snippets/{short_id}/delete", app.Sessions.RequireSession("/admin/login", app.adminDeleteSnippet))
	mux.HandleFunc("GET /s/{short_id}", app.viewSnippet)
	mux.HandleFunc("POST /snippets", app.createSnippetForm)

	mux.HandleFunc("GET /api/snippets", app.wrapIfAuth("api_list", app.apiList))
	mux.HandleFunc("POST /api/snippets", app.wrapIfAuth("api_create", app.apiCreate))
	mux.HandleFunc("GET /api/snippets/{short_id}", app.wrapIfAuth("api_get", app.apiGet))
	mux.HandleFunc("PUT /api/snippets/{short_id}", app.wrapIfAuth("api_update", app.apiUpdate))
	mux.HandleFunc("DELETE /api/snippets/{short_id}", app.wrapIfAuth("api_delete", app.apiDelete))

	mux.HandleFunc("GET /static/", web.EmbeddedHandler(appFS, "static"))
	darkmatter.Mount(mux, "/assets")

	addr := host + ":" + strconv.Itoa(port)
	logger.Info("sipp server running", "addr", addr)
	return http.ListenAndServe(addr, mux)
}

// Silence unused import warnings; keep these for forward use.
var _ = json.Marshal
var _ = bytes.NewReader
var _ io.Reader = (*bytes.Reader)(nil)
var _ = log.Fatal
