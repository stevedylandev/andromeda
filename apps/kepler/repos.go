package main

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

var defaultDescriptionPlaceholder = "Unnamed repository; edit this file 'description' to name the repository."

func (a *App) listRepos() ([]RepoSummary, error) {
	entries, err := os.ReadDir(a.RepoRoot)
	if err != nil {
		return nil, err
	}
	var out []RepoSummary
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path, name, ok := resolveRepoDir(a.RepoRoot, e.Name())
		if !ok {
			continue
		}
		s := RepoSummary{Name: name, Description: readDescription(path)}
		if repo, err := git.PlainOpen(path); err == nil {
			s.DefaultRef = defaultBranchName(repo)
			if head, err := repo.Head(); err == nil {
				if c, err := repo.CommitObject(head.Hash()); err == nil {
					s.LastCommit = c.Author.When
				}
			}
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].LastCommit.Equal(out[j].LastCommit) {
			return out[i].LastCommit.After(out[j].LastCommit)
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func (a *App) openRepo(name string) (*git.Repository, RepoSummary, error) {
	clean := strings.Trim(filepath.Clean(name), "/\\")
	if clean == "" || strings.Contains(clean, "..") || strings.ContainsAny(clean, "/\\") {
		return nil, RepoSummary{}, errors.New("invalid repo name")
	}
	path, resolvedName, ok := findRepoByName(a.RepoRoot, clean)
	if !ok {
		return nil, RepoSummary{}, errors.New("repo not found")
	}
	repo, err := git.PlainOpen(path)
	if err != nil {
		return nil, RepoSummary{}, err
	}
	s := RepoSummary{
		Name:        resolvedName,
		Description: readDescription(path),
		DefaultRef:  defaultBranchName(repo),
	}
	if head, err := repo.Head(); err == nil {
		if c, err := repo.CommitObject(head.Hash()); err == nil {
			s.LastCommit = c.Author.When
		}
	}
	return repo, s, nil
}

func resolveRepoDir(root, entry string) (path, name string, ok bool) {
	full := filepath.Join(root, entry)
	if strings.HasSuffix(entry, ".git") {
		if _, err := os.Stat(filepath.Join(full, "HEAD")); err == nil {
			return full, strings.TrimSuffix(entry, ".git"), true
		}
		return "", "", false
	}
	if _, err := os.Stat(filepath.Join(full, ".git", "HEAD")); err == nil {
		return full, entry, true
	}
	if _, err := os.Stat(filepath.Join(full, "HEAD")); err == nil {
		return full, entry, true
	}
	return "", "", false
}

func findRepoByName(root, name string) (string, string, bool) {
	candidates := []string{name + ".git", name}
	for _, c := range candidates {
		if p, n, ok := resolveRepoDir(root, c); ok {
			return p, n, true
		}
	}
	return "", "", false
}

func readDescription(repoPath string) string {
	data, err := os.ReadFile(filepath.Join(repoPath, "description"))
	if err != nil {
		return ""
	}
	s := strings.TrimSpace(string(data))
	if s == defaultDescriptionPlaceholder {
		return ""
	}
	return s
}

func defaultBranchName(repo *git.Repository) string {
	head, err := repo.Reference(plumbing.HEAD, false)
	if err != nil {
		return "main"
	}
	if head.Type() == plumbing.SymbolicReference {
		return head.Target().Short()
	}
	return "main"
}
