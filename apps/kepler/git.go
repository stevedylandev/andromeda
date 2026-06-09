package main

import (
	"errors"
	"io"
	"sort"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func resolveRef(repo *git.Repository, ref string) (*object.Commit, error) {
	if ref == "" {
		ref = defaultBranchName(repo)
	}
	hash, err := repo.ResolveRevision(plumbing.Revision(ref))
	if err != nil {
		return nil, err
	}
	return repo.CommitObject(*hash)
}

func listBranches(repo *git.Repository) ([]RefInfo, error) {
	iter, err := repo.Branches()
	if err != nil {
		return nil, err
	}
	var out []RefInfo
	_ = iter.ForEach(func(r *plumbing.Reference) error {
		info := RefInfo{Name: r.Name().Short(), SHA: r.Hash().String()}
		if c, err := repo.CommitObject(r.Hash()); err == nil {
			info.Time = c.Author.When
			info.Author = c.Author.Name
		}
		out = append(out, info)
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Time.After(out[j].Time) })
	return out, nil
}

func listTags(repo *git.Repository) ([]RefInfo, error) {
	iter, err := repo.Tags()
	if err != nil {
		return nil, err
	}
	var out []RefInfo
	_ = iter.ForEach(func(r *plumbing.Reference) error {
		info := RefInfo{Name: r.Name().Short(), SHA: r.Hash().String()}
		if tag, err := repo.TagObject(r.Hash()); err == nil {
			info.Time = tag.Tagger.When
			info.Author = tag.Tagger.Name
			if c, err := tag.Commit(); err == nil {
				info.SHA = c.Hash.String()
			}
		} else if c, err := repo.CommitObject(r.Hash()); err == nil {
			info.Time = c.Author.When
			info.Author = c.Author.Name
		}
		out = append(out, info)
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Time.After(out[j].Time) })
	return out, nil
}

func commitLog(repo *git.Repository, ref string, page, perPage int) ([]CommitInfo, bool, error) {
	head, err := resolveRef(repo, ref)
	if err != nil {
		return nil, false, err
	}
	iter, err := repo.Log(&git.LogOptions{From: head.Hash})
	if err != nil {
		return nil, false, err
	}
	defer iter.Close()

	skip := page * perPage
	var out []CommitInfo
	i := 0
	hasNext := false
	err = iter.ForEach(func(c *object.Commit) error {
		if i < skip {
			i++
			return nil
		}
		if len(out) >= perPage {
			hasNext = true
			return io.EOF
		}
		out = append(out, toCommitInfo(c))
		i++
		return nil
	})
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, false, err
	}
	return out, hasNext, nil
}

func toCommitInfo(c *object.Commit) CommitInfo {
	subject, body := splitCommitMessage(c.Message)
	parent := ""
	if c.NumParents() > 0 {
		parent = c.ParentHashes[0].String()
	}
	return CommitInfo{
		SHA:       c.Hash.String(),
		ShortSHA:  c.Hash.String()[:8],
		Author:    c.Author.Name,
		Email:     c.Author.Email,
		When:      c.Author.When,
		Subject:   subject,
		Body:      body,
		ParentSHA: parent,
	}
}

func splitCommitMessage(msg string) (subject, body string) {
	msg = strings.TrimRight(msg, "\n")
	if i := strings.Index(msg, "\n"); i >= 0 {
		return strings.TrimSpace(msg[:i]), strings.TrimSpace(msg[i+1:])
	}
	return msg, ""
}

func treeAt(repo *git.Repository, ref, path string) ([]TreeEntry, *object.Tree, error) {
	c, err := resolveRef(repo, ref)
	if err != nil {
		return nil, nil, err
	}
	root, err := c.Tree()
	if err != nil {
		return nil, nil, err
	}
	tree := root
	if path != "" {
		t, err := root.Tree(path)
		if err != nil {
			return nil, nil, err
		}
		tree = t
	}
	var out []TreeEntry
	for _, e := range tree.Entries {
		entryPath := e.Name
		if path != "" {
			entryPath = path + "/" + e.Name
		}
		te := TreeEntry{
			Name:  e.Name,
			Path:  entryPath,
			IsDir: e.Mode == filemode.Dir,
			Mode:  e.Mode.String(),
		}
		if !te.IsDir {
			if f, err := tree.File(e.Name); err == nil {
				te.Size = f.Size
			}
		}
		out = append(out, te)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].IsDir != out[j].IsDir {
			return out[i].IsDir
		}
		return out[i].Name < out[j].Name
	})
	return out, tree, nil
}

func blobAt(repo *git.Repository, ref, path string) (string, int64, bool, error) {
	c, err := resolveRef(repo, ref)
	if err != nil {
		return "", 0, false, err
	}
	tree, err := c.Tree()
	if err != nil {
		return "", 0, false, err
	}
	f, err := tree.File(path)
	if err != nil {
		return "", 0, false, err
	}
	bin, err := f.IsBinary()
	if err != nil {
		return "", 0, false, err
	}
	if bin {
		return "", f.Size, true, nil
	}
	content, err := f.Contents()
	if err != nil {
		return "", 0, false, err
	}
	return content, f.Size, false, nil
}

func blobBytes(repo *git.Repository, ref, path string) ([]byte, error) {
	c, err := resolveRef(repo, ref)
	if err != nil {
		return nil, err
	}
	tree, err := c.Tree()
	if err != nil {
		return nil, err
	}
	f, err := tree.File(path)
	if err != nil {
		return nil, err
	}
	r, err := f.Blob.Reader()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

func commitDiff(repo *git.Repository, sha string) (CommitInfo, []FilePatch, DiffStats, error) {
	h := plumbing.NewHash(sha)
	c, err := repo.CommitObject(h)
	if err != nil {
		return CommitInfo{}, nil, DiffStats{}, err
	}
	info := toCommitInfo(c)

	var parentTree *object.Tree
	if c.NumParents() > 0 {
		p, err := c.Parent(0)
		if err != nil {
			return info, nil, DiffStats{}, err
		}
		parentTree, err = p.Tree()
		if err != nil {
			return info, nil, DiffStats{}, err
		}
	}
	thisTree, err := c.Tree()
	if err != nil {
		return info, nil, DiffStats{}, err
	}
	patch, err := diffTrees(parentTree, thisTree)
	if err != nil {
		return info, nil, DiffStats{}, err
	}
	files, stats := convertPatch(patch)
	return info, files, stats, nil
}

func diffTrees(from, to *object.Tree) (*object.Patch, error) {
	if from == nil {
		return (&object.Tree{}).Patch(to)
	}
	return from.Patch(to)
}
