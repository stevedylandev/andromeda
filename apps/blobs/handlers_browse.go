package main

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
)

func (a *App) listBuckets(w http.ResponseWriter, r *http.Request) {
	buckets, err := a.S3.ListBuckets(r.Context())
	data := bucketsPageData{Buckets: buckets}
	if err != nil {
		a.Log.Error("list buckets", "err", err)
		data.Error = err.Error()
	}
	a.renderPage(w, "buckets.html", data)
}

func (a *App) bucketRoot(w http.ResponseWriter, r *http.Request) {
	bucket := r.PathValue("bucket")
	http.Redirect(w, r, browseHref(bucket, ""), http.StatusSeeOther)
}

func (a *App) browse(w http.ResponseWriter, r *http.Request) {
	bucket := r.PathValue("bucket")
	prefix := r.PathValue("prefix")
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		http.Redirect(w, r, browseHref(bucket, prefix+"/"), http.StatusSeeOther)
		return
	}

	folders, files, err := a.S3.List(r.Context(), bucket, prefix)
	data := browsePageData{
		Bucket:      bucket,
		Prefix:      prefix,
		Crumbs:      buildCrumbs(bucket, prefix),
		Error:       r.URL.Query().Get("error"),
		Success:     r.URL.Query().Get("success"),
		MaxUploadMB: a.MaxUploadBytes >> 20,
	}
	if prefix != "" {
		data.ParentHref = browseHref(bucket, parentPrefix(prefix))
	}
	if err != nil {
		a.Log.Error("list", "bucket", bucket, "prefix", prefix, "err", err)
		if data.Error == "" {
			data.Error = err.Error()
		}
	}
	for _, f := range folders {
		name := strings.TrimSuffix(strings.TrimPrefix(f, prefix), "/")
		if name == "" {
			continue
		}
		data.Folders = append(data.Folders, folderItem{
			Name: name,
			Href: browseHref(bucket, f),
		})
	}
	for _, f := range files {
		// Skip the zero-byte "folder marker" key that matches the prefix exactly.
		if f.Name == "" {
			continue
		}
		img := isImageName(f.Name)
		item := fileItem{
			Name:         f.Name,
			Size:         f.Size,
			SizeHuman:    humanSize(f.Size),
			LastModified: f.LastModified,
			DetailHref:   objectHref(bucket, f.Key),
			IsImage:      img,
		}
		if img {
			item.PreviewSrc = previewHref(bucket, f.Key)
		}
		data.Files = append(data.Files, item)
	}
	a.renderPage(w, "browse.html", data)
}

func (a *App) detail(w http.ResponseWriter, r *http.Request) {
	bucket := r.PathValue("bucket")
	key := r.PathValue("key")
	if key == "" {
		http.NotFound(w, r)
		return
	}
	meta, err := a.S3.Head(r.Context(), bucket, key)
	if err != nil {
		a.Log.Error("head", "bucket", bucket, "key", key, "err", err)
		http.Error(w, "object not found", http.StatusNotFound)
		return
	}
	parent := parentOfKey(key)
	data := detailPageData{
		Bucket:       bucket,
		Key:          key,
		Name:         nameOfKey(key),
		ContentType:  meta.ContentType,
		Size:         meta.Size,
		SizeHuman:    humanSize(meta.Size),
		LastModified: meta.LastModified,
		ETag:         meta.ETag,
		IsImage:      isImageName(key) || strings.HasPrefix(meta.ContentType, "image/"),
		PreviewSrc:   previewHref(bucket, key),
		ParentHref:   browseHref(bucket, parent),
		ParentPrefix: parent,
		Crumbs:       buildCrumbs(bucket, parent),
		Error:        r.URL.Query().Get("error"),
	}
	if u, perr := a.S3.PresignGet(r.Context(), bucket, key); perr == nil {
		data.PresignedURL = u
	} else {
		a.Log.Warn("presign", "err", perr)
	}
	if u, ok := a.S3.PublicURL(bucket, key); ok {
		data.PublicURL = u
		data.HasPublic = true
	}
	a.renderPage(w, "detail.html", data)
}

func (a *App) preview(w http.ResponseWriter, r *http.Request) {
	bucket := r.PathValue("bucket")
	key := r.PathValue("key")
	if key == "" {
		http.NotFound(w, r)
		return
	}
	body, meta, err := a.S3.Get(r.Context(), bucket, key)
	if err != nil {
		a.Log.Error("get", "bucket", bucket, "key", key, "err", err)
		http.Error(w, "object not found", http.StatusNotFound)
		return
	}
	defer body.Close()
	if meta.ContentType != "" {
		w.Header().Set("Content-Type", meta.ContentType)
	}
	if meta.Size > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(meta.Size, 10))
	}
	if meta.ETag != "" {
		w.Header().Set("ETag", `"`+meta.ETag+`"`)
	}
	w.Header().Set("Cache-Control", "private, max-age=60")
	if _, err := io.Copy(w, body); err != nil && !errors.Is(err, http.ErrBodyReadAfterClose) {
		a.Log.Warn("preview copy", "err", err)
	}
}
