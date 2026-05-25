package main

import (
	"context"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"
)

func parseUploadArgs(args []string) (ClientFlags, string) {
	var (
		flags ClientFlags
		file  string
	)
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case (a == "-b" || a == "--bucket") && i+1 < len(args):
			flags.Bucket = args[i+1]
			i++
		case strings.HasPrefix(a, "--bucket="):
			flags.Bucket = strings.TrimPrefix(a, "--bucket=")
		case a == "--prefix" && i+1 < len(args):
			flags.Prefix = args[i+1]
			i++
		case strings.HasPrefix(a, "--prefix="):
			flags.Prefix = strings.TrimPrefix(a, "--prefix=")
		case a == "--key" && i+1 < len(args):
			flags.Key = args[i+1]
			i++
		case strings.HasPrefix(a, "--key="):
			flags.Key = strings.TrimPrefix(a, "--key=")
		case !strings.HasPrefix(a, "-") && file == "":
			file = a
		}
	}
	return flags, file
}

func runUpload(args []string) {
	flags, file := parseUploadArgs(args)
	if file == "" {
		fmt.Fprintln(os.Stderr, "no file specified")
		os.Exit(2)
	}

	cfg, err := LoadClientConfig(flags)
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}
	bucket := cfg.DefaultBucket
	if bucket == "" {
		fmt.Fprintln(os.Stderr, "no bucket: pass -b/--bucket or set BLOBS_DEFAULT_BUCKET")
		os.Exit(2)
	}

	s3, err := NewS3FromConfig(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "s3:", err)
		os.Exit(1)
	}

	f, err := os.Open(file)
	if err != nil {
		fmt.Fprintln(os.Stderr, "open:", err)
		os.Exit(1)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		fmt.Fprintln(os.Stderr, "stat:", err)
		os.Exit(1)
	}

	name := flags.Key
	if name == "" {
		name = filepath.Base(file)
	}
	prefix := flags.Prefix
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	key := prefix + name

	ct := mime.TypeByExtension(filepath.Ext(name))
	if ct == "" {
		ct = "application/octet-stream"
	}

	ctx := context.Background()
	if err := s3.Put(ctx, bucket, key, ct, f, info.Size()); err != nil {
		fmt.Fprintln(os.Stderr, "upload:", err)
		os.Exit(1)
	}

	url, err := ResolveURL(ctx, s3, bucket, key)
	if err != nil {
		fmt.Fprintln(os.Stderr, "url:", err)
		fmt.Println(key)
		return
	}
	fmt.Println(url)
}
