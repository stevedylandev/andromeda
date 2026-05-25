package main

import (
	"bytes"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/stevedylandev/andromeda/pkg/web"
)

func (a *App) upload(w http.ResponseWriter, r *http.Request) {
	bucket := r.PathValue("bucket")
	r.Body = http.MaxBytesReader(w, r.Body, a.MaxUploadBytes+1<<20)
	if err := r.ParseMultipartForm(a.MaxUploadBytes); err != nil {
		web.RedirectWithError(w, r, browseHref(bucket, ""), "Upload too large or malformed")
		return
	}
	prefix := strings.TrimSpace(r.FormValue("prefix"))
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	target := browseHref(bucket, prefix)

	files := r.MultipartForm.File["file"]
	if len(files) == 0 {
		web.RedirectWithError(w, r, target, "No file provided")
		return
	}
	for _, header := range files {
		if header.Size > a.MaxUploadBytes {
			web.RedirectWithError(w, r, target, "File exceeds upload limit")
			return
		}
		f, err := header.Open()
		if err != nil {
			web.RedirectWithError(w, r, target, "Failed to read upload")
			return
		}
		name := path.Base(header.Filename)
		if name == "" || name == "." || name == "/" {
			f.Close()
			web.RedirectWithError(w, r, target, "Invalid filename")
			return
		}
		ct := header.Header.Get("Content-Type")
		if ct == "" {
			ct = mime.TypeByExtension(path.Ext(name))
		}
		if ct == "" {
			ct = "application/octet-stream"
		}
		if err := a.S3.Put(r.Context(), bucket, prefix+name, ct, f, header.Size); err != nil {
			f.Close()
			a.Log.Error("put", "bucket", bucket, "key", prefix+name, "err", err)
			web.RedirectWithError(w, r, target, "Upload failed: "+err.Error())
			return
		}
		f.Close()
	}
	web.RedirectWithSuccess(w, r, target, "Uploaded")
}

func (a *App) replace(w http.ResponseWriter, r *http.Request) {
	bucket := r.PathValue("bucket")
	r.Body = http.MaxBytesReader(w, r.Body, a.MaxUploadBytes+1<<20)
	if err := r.ParseMultipartForm(a.MaxUploadBytes); err != nil {
		web.RedirectWithError(w, r, browseHref(bucket, ""), "Upload too large or malformed")
		return
	}
	key := strings.TrimSpace(r.FormValue("key"))
	if key == "" {
		http.Error(w, "key required", http.StatusBadRequest)
		return
	}
	target := "/b/" + url.PathEscape(bucket) + "/object/" + escapeKeyPath(key)
	f, header, err := r.FormFile("file")
	if err != nil {
		web.RedirectWithError(w, r, target, "No file provided")
		return
	}
	defer f.Close()
	if header.Size > a.MaxUploadBytes {
		web.RedirectWithError(w, r, target, "File exceeds upload limit")
		return
	}
	ct := header.Header.Get("Content-Type")
	if ct == "" {
		ct = mime.TypeByExtension(path.Ext(key))
	}
	if ct == "" {
		ct = "application/octet-stream"
	}
	if err := a.S3.Put(r.Context(), bucket, key, ct, f, header.Size); err != nil {
		a.Log.Error("put replace", "bucket", bucket, "key", key, "err", err)
		web.RedirectWithError(w, r, target, "Replace failed: "+err.Error())
		return
	}
	web.RedirectWithSuccess(w, r, target, "Replaced")
}

func (a *App) deleteObject(w http.ResponseWriter, r *http.Request) {
	bucket := r.PathValue("bucket")
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	key := strings.TrimSpace(r.FormValue("key"))
	if key == "" {
		http.Error(w, "key required", http.StatusBadRequest)
		return
	}
	returnTo := strings.TrimSpace(r.FormValue("returnTo"))
	if returnTo == "" {
		returnTo = browseHref(bucket, parentOfKey(strings.TrimSuffix(key, "/")))
	}
	if err := a.S3.Delete(r.Context(), bucket, key); err != nil {
		a.Log.Error("delete", "bucket", bucket, "key", key, "err", err)
		web.RedirectWithError(w, r, returnTo, "Delete failed: "+err.Error())
		return
	}
	web.RedirectWithSuccess(w, r, returnTo, "Deleted")
}

func (a *App) mkdir(w http.ResponseWriter, r *http.Request) {
	bucket := r.PathValue("bucket")
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	prefix := strings.TrimSpace(r.FormValue("prefix"))
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	name := strings.TrimSpace(r.FormValue("name"))
	name = strings.Trim(name, "/")
	if name == "" || strings.HasPrefix(name, ".") || strings.Contains(name, "/") {
		web.RedirectWithError(w, r, browseHref(bucket, prefix), "Invalid folder name")
		return
	}
	newPrefix := prefix + name + "/"
	if err := a.S3.Put(r.Context(), bucket, newPrefix, "application/x-directory", bytes.NewReader(nil), 0); err != nil {
		a.Log.Error("mkdir", "bucket", bucket, "key", newPrefix, "err", err)
		web.RedirectWithError(w, r, browseHref(bucket, prefix), "Create folder failed: "+err.Error())
		return
	}
	web.RedirectWithSuccess(w, r, browseHref(bucket, newPrefix), "Folder created")
}
