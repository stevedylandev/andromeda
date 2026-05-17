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

	"github.com/stevedylandev/andromeda/apps/jotts-go/internal/store"
	"github.com/stevedylandev/andromeda/crates-go/config"
)

type Note = store.Note

type Backend interface {
	List() ([]Note, error)
	Get(shortID string) (*Note, error)
	Create(title, content string) (*Note, error)
	Update(shortID, title, content string) (*Note, error)
	Delete(shortID string) (bool, error)
	RemoteURL() string
	Close() error
}

type LocalBackend struct {
	DB *sql.DB
}

func (b *LocalBackend) List() ([]Note, error)             { return store.List(b.DB) }
func (b *LocalBackend) Get(s string) (*Note, error)       { return store.GetByShortID(b.DB, s) }
func (b *LocalBackend) Create(t, c string) (*Note, error) { return store.Create(b.DB, t, c) }
func (b *LocalBackend) Update(s, t, c string) (*Note, error) {
	return store.UpdateByShortID(b.DB, s, t, c)
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

func (r *RemoteBackend) List() ([]Note, error) {
	var out []Note
	if err := r.do("GET", "/api/notes", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *RemoteBackend) Get(shortID string) (*Note, error) {
	var n Note
	if err := r.do("GET", "/api/notes/"+shortID, nil, &n); err != nil {
		if err == errNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &n, nil
}

func (r *RemoteBackend) Create(title, content string) (*Note, error) {
	var n Note
	if err := r.do("POST", "/api/notes", store.NoteInput{Title: title, Content: content}, &n); err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *RemoteBackend) Update(shortID, title, content string) (*Note, error) {
	var n Note
	if err := r.do("PUT", "/api/notes/"+shortID, store.NoteInput{Title: title, Content: content}, &n); err != nil {
		if err == errNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &n, nil
}

func (r *RemoteBackend) Delete(shortID string) (bool, error) {
	if err := r.do("DELETE", "/api/notes/"+shortID, nil, nil); err != nil {
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
		case a == "--remote" && i+1 < len(args):
			opts.RemoteURL = args[i+1]
			i++
		case strings.HasPrefix(a, "--remote="):
			opts.RemoteURL = strings.TrimPrefix(a, "--remote=")
		case a == "--api-key" && i+1 < len(args):
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
	explicitRemote := remoteURL != ""
	if remoteURL == "" {
		remoteURL = os.Getenv("JOTTS_REMOTE_URL")
	}
	if remoteURL == "" {
		remoteURL = cfg.RemoteURL
	}

	apiKey := opts.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("JOTTS_API_KEY")
	}
	if apiKey == "" {
		apiKey = cfg.APIKey
	}

	dbPath := opts.DBPath
	explicitDB := dbPath != ""
	if dbPath == "" {
		dbPath = config.Getenv("JOTTS_DB_PATH", "jotts.sqlite")
	}

	useRemote := remoteURL != "" && (!explicitDB || explicitRemote)

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
