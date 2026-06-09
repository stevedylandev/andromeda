package main

import (
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
)

const commitsPerPage = 30

func (a *App) base(repoName string) pageBase {
	return pageBase{SiteName: a.SiteName, RepoName: repoName}
}

func (a *App) indexHandler(w http.ResponseWriter, r *http.Request) {
	repos, err := a.listRepos()
	if err != nil {
		a.Log.Error("list repos failed", "err", err)
		http.Error(w, "failed to list repos", http.StatusInternalServerError)
		return
	}
	a.renderPage(w, "index.html", indexPageData{pageBase: a.base(""), Repos: repos})
}

func (a *App) repoHandler(w http.ResponseWriter, r *http.Request) {
	repo, summary, err := a.openRepo(r.PathValue("repo"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	branches, _ := listBranches(repo)
	tags, _ := listTags(repo)
	commits, _, _ := commitLog(repo, summary.DefaultRef, 0, 1)
	entries, _, _ := treeAt(repo, summary.DefaultRef, "")

	var latest CommitInfo
	if len(commits) > 0 {
		latest = commits[0]
	}

	data := repoPageData{
		pageBase:     a.base(summary.Name),
		Repo:         summary,
		Ref:          summary.DefaultRef,
		DefaultRef:   summary.DefaultRef,
		Branches:     branches,
		Tags:         tags,
		Commits:      commits,
		LatestCommit: latest,
		HasLatest:    len(commits) > 0,
		Entries:      entries,
	}

	if c, err := resolveRef(repo, summary.DefaultRef); err == nil {
		if tree, err := c.Tree(); err == nil {
			if src, ok := findReadme(tree); ok {
				rawBase := "/" + summary.Name + "/raw/" + summary.DefaultRef + "/"
				if html, err := renderMarkdown(src, rawBase); err == nil {
					data.ReadmeHTML = html
					data.HasReadme = true
				}
			}
		}
	}

	a.renderPage(w, "repo.html", data)
}

func (a *App) treeHandler(w http.ResponseWriter, r *http.Request) {
	repo, summary, err := a.openRepo(r.PathValue("repo"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	ref := r.PathValue("ref")
	subPath := strings.Trim(r.PathValue("path"), "/")

	entries, _, err := treeAt(repo, ref, subPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	data := treePageData{
		pageBase:    a.base(summary.Name),
		Repo:        summary,
		Ref:         ref,
		DefaultRef:  summary.DefaultRef,
		Path:        subPath,
		Breadcrumbs: makeBreadcrumbs(summary.Name, ref, subPath),
		Entries:     entries,
	}
	a.renderPage(w, "tree.html", data)
}

func (a *App) blobHandler(w http.ResponseWriter, r *http.Request) {
	repo, summary, err := a.openRepo(r.PathValue("repo"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	ref := r.PathValue("ref")
	subPath := strings.Trim(r.PathValue("path"), "/")

	content, size, isBin, err := blobAt(repo, ref, subPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	data := blobPageData{
		pageBase:    a.base(summary.Name),
		Repo:        summary,
		Ref:         ref,
		DefaultRef:  summary.DefaultRef,
		Path:        subPath,
		Breadcrumbs: makeBreadcrumbs(summary.Name, ref, subPath),
		Binary:      isBin,
		Size:        size,
	}
	if !isBin {
		data.HighlightedHTML = renderBlobLines(content)
	}
	a.renderPage(w, "blob.html", data)
}

func (a *App) rawHandler(w http.ResponseWriter, r *http.Request) {
	repo, _, err := a.openRepo(r.PathValue("repo"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	ref := r.PathValue("ref")
	subPath := strings.Trim(r.PathValue("path"), "/")

	data, err := blobBytes(repo, ref, subPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	ct := http.DetectContentType(data)
	// http.DetectContentType can't sniff SVG (returns text/xml or
	// text/plain), which browsers won't render in <img>. Trust the
	// extension for types the sniffer gets wrong.
	if ext := strings.ToLower(path.Ext(subPath)); ext != "" {
		if byExt := mime.TypeByExtension(ext); byExt != "" {
			ct = byExt
		}
	}
	w.Header().Set("Content-Type", ct)
	_, _ = w.Write(data)
}

func (a *App) logHandler(w http.ResponseWriter, r *http.Request) {
	repo, summary, err := a.openRepo(r.PathValue("repo"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	ref := r.PathValue("ref")
	page := 0
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}
	commits, hasNext, err := commitLog(repo, ref, page, commitsPerPage)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	a.renderPage(w, "log.html", logPageData{
		pageBase:   a.base(summary.Name),
		Repo:       summary,
		Ref:        ref,
		DefaultRef: summary.DefaultRef,
		Commits:    commits,
		Page:       page,
		NextPage:   page + 1,
		PrevPage:   page - 1,
		HasNext:    hasNext,
		HasPrev:    page > 0,
	})
}

func (a *App) commitHandler(w http.ResponseWriter, r *http.Request) {
	repo, summary, err := a.openRepo(r.PathValue("repo"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	sha := r.PathValue("sha")
	info, files, stats, err := commitDiff(repo, sha)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	a.renderPage(w, "commit.html", commitPageData{
		pageBase:   a.base(summary.Name),
		Repo:       summary,
		DefaultRef: summary.DefaultRef,
		Commit:     info,
		Files:      files,
		Stats:      stats,
	})
}

func (a *App) refsHandler(w http.ResponseWriter, r *http.Request) {
	repo, summary, err := a.openRepo(r.PathValue("repo"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	branches, _ := listBranches(repo)
	tags, _ := listTags(repo)
	a.renderPage(w, "refs.html", refsPageData{
		pageBase:   a.base(summary.Name),
		Repo:       summary,
		DefaultRef: summary.DefaultRef,
		Branches:   branches,
		Tags:       tags,
	})
}

func (a *App) archiveHandler(w http.ResponseWriter, r *http.Request) {
	repo, summary, err := a.openRepo(r.PathValue("repo"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	ref, format, ok := parseArchiveName(r.PathValue("name"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	prefix := summary.Name + "-" + ref
	switch format {
	case "tar.gz":
		w.Header().Set("Content-Type", "application/gzip")
		w.Header().Set("Content-Disposition", `attachment; filename="`+summary.Name+"-"+ref+`.tar.gz"`)
		if err := writeTarGz(w, repo, ref, prefix); err != nil {
			a.Log.Error("tar.gz failed", "err", err)
		}
	case "zip":
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", `attachment; filename="`+summary.Name+"-"+ref+`.zip"`)
		if err := writeZip(w, repo, ref, prefix); err != nil {
			a.Log.Error("zip failed", "err", err)
		}
	}
}

func (a *App) atomHandler(w http.ResponseWriter, r *http.Request) {
	repo, summary, err := a.openRepo(r.PathValue("repo"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	commits, _, err := commitLog(repo, summary.DefaultRef, 0, 20)
	if err != nil {
		http.Error(w, "log failed", http.StatusInternalServerError)
		return
	}
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	baseURL := scheme + "://" + r.Host
	feed, err := buildAtomFeed(a.SiteName, summary.Name, baseURL, commits)
	if err != nil {
		http.Error(w, "atom build failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
	_, _ = w.Write(feed)
}

func makeBreadcrumbs(repoName, ref, subPath string) []Breadcrumb {
	out := []Breadcrumb{{Name: repoName, Href: "/" + repoName + "/tree/" + ref}}
	if subPath == "" {
		return out
	}
	parts := strings.Split(subPath, "/")
	acc := ""
	for _, p := range parts {
		if acc == "" {
			acc = p
		} else {
			acc = acc + "/" + p
		}
		out = append(out, Breadcrumb{Name: p, Href: "/" + repoName + "/tree/" + ref + "/" + acc})
	}
	return out
}
