package tui

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/stevedylandev/andromeda/apps/sipp-go/internal/store"
	"github.com/stevedylandev/andromeda/crates-go/config"
)

type Snippet = store.Snippet

type Backend interface {
	List() ([]Snippet, error)
	Get(shortID string) (*Snippet, error)
	Create(name, content string) (*Snippet, error)
	Update(shortID, name, content string) (*Snippet, error)
	Delete(shortID string) (bool, error)
	RemoteURL() string
	Close() error
}

type LocalBackend struct {
	DB *sql.DB
}

func (b *LocalBackend) List() ([]Snippet, error)             { return store.List(b.DB) }
func (b *LocalBackend) Get(s string) (*Snippet, error)       { return store.GetByShortID(b.DB, s) }
func (b *LocalBackend) Create(n, c string) (*Snippet, error) { return store.Create(b.DB, n, c) }
func (b *LocalBackend) Update(s, n, c string) (*Snippet, error) {
	return store.UpdateByShortID(b.DB, s, n, c)
}
func (b *LocalBackend) Delete(s string) (bool, error) { return store.DeleteByShortID(b.DB, s) }
func (b *LocalBackend) RemoteURL() string             { return "" }
func (b *LocalBackend) Close() error                  { return b.DB.Close() }

type RemoteBackend struct {
	BaseURL string
	APIKey  string
	Client  *http.Client
}

func (r *RemoteBackend) RemoteURL() string { return r.BaseURL }
func (r *RemoteBackend) Close() error      { return nil }

func (r *RemoteBackend) do(method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(buf)
	}
	req, err := http.NewRequest(method, strings.TrimRight(r.BaseURL, "/")+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if r.APIKey != "" {
		req.Header.Set("x-api-key", r.APIKey)
	}
	resp, err := r.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return errNotFound
	}
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	if out == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

var errNotFound = fmt.Errorf("not found")

func (r *RemoteBackend) List() ([]Snippet, error) {
	var out []Snippet
	if err := r.do("GET", "/api/snippets", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *RemoteBackend) Get(shortID string) (*Snippet, error) {
	var s Snippet
	if err := r.do("GET", "/api/snippets/"+shortID, nil, &s); err != nil {
		if err == errNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *RemoteBackend) Create(name, content string) (*Snippet, error) {
	var s Snippet
	if err := r.do("POST", "/api/snippets", store.SnippetInput{Name: name, Content: content}, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *RemoteBackend) Update(shortID, name, content string) (*Snippet, error) {
	var s Snippet
	if err := r.do("PUT", "/api/snippets/"+shortID, store.SnippetInput{Name: name, Content: content}, &s); err != nil {
		if err == errNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *RemoteBackend) Delete(shortID string) (bool, error) {
	if err := r.do("DELETE", "/api/snippets/"+shortID, nil, nil); err != nil {
		if err == errNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

type Options struct {
	RemoteURL string
	APIKey    string
	DBPath    string
}

func ParseArgs(args []string) Options {
	opts := Options{}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case (a == "--remote" || a == "-r") && i+1 < len(args):
			opts.RemoteURL = args[i+1]
			i++
		case strings.HasPrefix(a, "--remote="):
			opts.RemoteURL = strings.TrimPrefix(a, "--remote=")
		case (a == "--api-key" || a == "-k") && i+1 < len(args):
			opts.APIKey = args[i+1]
			i++
		case strings.HasPrefix(a, "--api-key="):
			opts.APIKey = strings.TrimPrefix(a, "--api-key=")
		case a == "--db" && i+1 < len(args):
			opts.DBPath = args[i+1]
			i++
		}
	}
	return opts
}

func ResolveBackend(opts Options) (Backend, error) {
	cfg, _ := LoadConfig()

	remoteURL := opts.RemoteURL
	if remoteURL == "" {
		remoteURL = os.Getenv("SIPP_REMOTE_URL")
	}
	apiKey := opts.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("SIPP_API_KEY")
	}
	if apiKey == "" {
		apiKey = cfg.APIKey
	}

	dbPath := opts.DBPath
	if dbPath == "" {
		dbPath = config.Getenv("SIPP_DB_PATH", "sipp.sqlite")
	}

	useRemote := remoteURL != ""
	if !useRemote {
		if _, err := os.Stat(dbPath); err != nil && cfg.RemoteURL != "" {
			remoteURL = cfg.RemoteURL
			useRemote = true
		}
	}

	if useRemote {
		return &RemoteBackend{
			BaseURL: remoteURL,
			APIKey:  apiKey,
			Client:  &http.Client{Timeout: 15 * time.Second},
		}, nil
	}

	db, err := store.Open(dbPath)
	if err != nil {
		return nil, err
	}
	return &LocalBackend{DB: db}, nil
}
