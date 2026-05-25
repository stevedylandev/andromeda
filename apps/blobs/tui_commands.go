package main

import (
	"context"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/stevedylandev/andromeda/apps/blobs/preview"
)

func loadBucketsCmd(s3 *S3Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		bs, err := s3.ListBuckets(ctx)
		return bucketsLoadedMsg{Buckets: bs, Err: err}
	}
}

func loadListingCmd(s3 *S3Client, bucket, prefix string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		folders, files, err := s3.List(ctx, bucket, prefix)
		return listingLoadedMsg{Bucket: bucket, Prefix: prefix, Folders: folders, Files: files, Err: err}
	}
}

const previewMaxBytes = 5 << 20

func loadPreviewCmd(s3 *S3Client, proto preview.Protocol, bucket, key string, w, h int) tea.Cmd {
	return func() tea.Msg {
		if !isImageName(key) {
			return previewLoadedMsg{Bucket: bucket, Key: key, Content: previewMetaText(key, 0, "")}
		}
		if proto == preview.ProtoNone {
			return previewLoadedMsg{Bucket: bucket, Key: key, Content: previewMetaText(key, 0, "no preview backend")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		body, meta, err := s3.Get(ctx, bucket, key)
		if err != nil {
			return previewLoadedMsg{Bucket: bucket, Key: key, Err: err}
		}
		defer body.Close()
		if meta != nil && meta.Size > previewMaxBytes {
			return previewLoadedMsg{Bucket: bucket, Key: key, Content: previewMetaText(key, meta.Size, "too large to preview")}
		}
		buf, err := io.ReadAll(io.LimitReader(body, previewMaxBytes+1))
		if err != nil {
			return previewLoadedMsg{Bucket: bucket, Key: key, Err: err}
		}
		if len(buf) > previewMaxBytes {
			return previewLoadedMsg{Bucket: bucket, Key: key, Content: previewMetaText(key, int64(len(buf)), "too large to preview")}
		}
		rendered, rerr := preview.Render(proto, buf, w, h)
		if rerr != nil {
			return previewLoadedMsg{Bucket: bucket, Key: key, Content: previewMetaText(key, int64(len(buf)), "render: "+rerr.Error())}
		}
		return previewLoadedMsg{Bucket: bucket, Key: key, Content: rendered}
	}
}

func previewMetaText(key string, size int64, note string) string {
	ct := mime.TypeByExtension(filepath.Ext(key))
	if ct == "" {
		ct = "application/octet-stream"
	}
	out := fmt.Sprintf("key:  %s\ntype: %s", key, ct)
	if size > 0 {
		out += fmt.Sprintf("\nsize: %s", humanSize(size))
	}
	if note != "" {
		out += "\n\n" + note
	}
	return out
}

func deleteCmd(s3 *S3Client, bucket, key string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		err := s3.Delete(ctx, bucket, key)
		return deletedMsg{Key: key, Err: err}
	}
}

func uploadCmd(s3 *S3Client, cfg ClientConfig, bucket, prefix, localPath string) tea.Cmd {
	return func() tea.Msg {
		f, err := os.Open(localPath)
		if err != nil {
			return uploadedMsg{Err: err}
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil {
			return uploadedMsg{Err: err}
		}
		name := filepath.Base(localPath)
		pfx := prefix
		if pfx != "" && !strings.HasSuffix(pfx, "/") {
			pfx += "/"
		}
		key := pfx + name
		ct := mime.TypeByExtension(filepath.Ext(name))
		if ct == "" {
			ct = "application/octet-stream"
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := s3.Put(ctx, bucket, key, ct, f, info.Size()); err != nil {
			return uploadedMsg{Key: key, Err: err}
		}
		url, _ := ResolveURL(ctx, s3, bucket, key)
		return uploadedMsg{Key: key, URL: url}
	}
}
