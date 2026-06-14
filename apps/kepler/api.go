package main

import (
	"encoding/json"
	"net/http"
	"time"
)

type apiRepo struct {
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	DefaultRef  string    `json:"default_ref,omitempty"`
	LastCommit  time.Time `json:"last_commit,omitempty"`
	URL         string    `json:"url"`
	AtomURL     string    `json:"atom_url"`
	CloneHTTPS  string    `json:"clone_https,omitempty"`
	CloneSSH    string    `json:"clone_ssh,omitempty"`
}

type apiReposResponse struct {
	Site  string    `json:"site"`
	Repos []apiRepo `json:"repos"`
}

func (a *App) apiReposHandler(w http.ResponseWriter, r *http.Request) {
	repos, err := a.listRepos()
	if err != nil {
		a.Log.Error("list repos failed", "err", err)
		http.Error(w, "failed to list repos", http.StatusInternalServerError)
		return
	}

	baseURL := a.requestBaseURL(r)
	resp := apiReposResponse{Site: a.SiteName, Repos: make([]apiRepo, 0, len(repos))}
	for _, s := range repos {
		ar := apiRepo{
			Name:        s.Name,
			Description: s.Description,
			DefaultRef:  s.DefaultRef,
			LastCommit:  s.LastCommit,
			URL:         baseURL + "/" + s.Name,
			AtomURL:     baseURL + "/" + s.Name + "/atom.xml",
		}
		if a.CloneBaseURL != "" {
			ar.CloneHTTPS = a.CloneBaseURL + "/" + s.Name + ".git"
		}
		if a.CloneSSHHost != "" {
			ar.CloneSSH = a.CloneSSHHost + ":" + s.Name + ".git"
		}
		resp.Repos = append(resp.Repos, ar)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(resp); err != nil {
		a.Log.Error("api repos encode failed", "err", err)
	}
}
