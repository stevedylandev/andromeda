package main

import (
	"database/sql"
	"embed"
	"html/template"
	"log/slog"

	"github.com/stevedylandev/andromeda/pkg/auth"
)

//go:embed templates/*.html static/*
var appFS embed.FS

type App struct {
	DB             *sql.DB
	Log            *slog.Logger
	Templates      map[string]*template.Template
	Sessions       *auth.Store
	S3             *S3Client
	Password       string
	CookieSecure   bool
	MaxUploadBytes int64
}

type loginPageData struct {
	Error string
}

type bucketsPageData struct {
	Buckets []BucketInfo
	Error   string
}

type crumb struct {
	Label string
	Href  string
}

type folderItem struct {
	Name string
	Href string
}

type fileItem struct {
	Name         string
	Size         int64
	SizeHuman    string
	LastModified string
	DetailHref   string
	PreviewSrc   string
	IsImage      bool
}

type browsePageData struct {
	Bucket   string
	Prefix   string
	ParentHref string
	Crumbs   []crumb
	Folders  []folderItem
	Files    []fileItem
	Error    string
	Success  string
	MaxUploadMB int64
}

type detailPageData struct {
	Bucket       string
	Key          string
	Name         string
	ContentType  string
	Size         int64
	SizeHuman    string
	LastModified string
	ETag         string
	IsImage      bool
	PreviewSrc   string
	PresignedURL string
	PublicURL    string
	HasPublic    bool
	ParentHref   string
	ParentPrefix string
	Crumbs       []crumb
	Error        string
}
