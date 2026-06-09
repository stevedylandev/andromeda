package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func writeTarGz(w http.ResponseWriter, repo *git.Repository, ref, prefix string) error {
	c, err := resolveRef(repo, ref)
	if err != nil {
		return err
	}
	tree, err := c.Tree()
	if err != nil {
		return err
	}

	gz := gzip.NewWriter(w)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()

	return tree.Files().ForEach(func(f *object.File) error {
		mode := int64(0644)
		if f.Mode == filemode.Executable {
			mode = 0755
		}
		hdr := &tar.Header{
			Name: prefix + "/" + f.Name,
			Mode: mode,
			Size: f.Size,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		r, err := f.Blob.Reader()
		if err != nil {
			return err
		}
		_, err = io.Copy(tw, r)
		r.Close()
		return err
	})
}

func writeZip(w http.ResponseWriter, repo *git.Repository, ref, prefix string) error {
	c, err := resolveRef(repo, ref)
	if err != nil {
		return err
	}
	tree, err := c.Tree()
	if err != nil {
		return err
	}

	zw := zip.NewWriter(w)
	defer zw.Close()

	return tree.Files().ForEach(func(f *object.File) error {
		zf, err := zw.Create(prefix + "/" + f.Name)
		if err != nil {
			return err
		}
		r, err := f.Blob.Reader()
		if err != nil {
			return err
		}
		_, err = io.Copy(zf, r)
		r.Close()
		return err
	})
}

func parseArchiveName(name string) (ref, format string, ok bool) {
	if strings.HasSuffix(name, ".tar.gz") {
		return strings.TrimSuffix(name, ".tar.gz"), "tar.gz", true
	}
	if strings.HasSuffix(name, ".tgz") {
		return strings.TrimSuffix(name, ".tgz"), "tar.gz", true
	}
	if strings.HasSuffix(name, ".zip") {
		return strings.TrimSuffix(name, ".zip"), "zip", true
	}
	return "", "", false
}
