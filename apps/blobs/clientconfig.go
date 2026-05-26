package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/stevedylandev/andromeda/pkg/config"
)

// ClientConfig is the on-disk client config for the blobs TUI/CLI.
// Server still reads from env/.env directly; this is purely for client use
// when running outside the server's working directory.
type ClientConfig struct {
	Endpoint        string            `toml:"endpoint"`
	Region          string            `toml:"region"`
	AccessKeyID     string            `toml:"access_key_id"`
	SecretAccessKey string            `toml:"secret_access_key"`
	R2AccountID     string            `toml:"r2_account_id,omitempty"`
	DefaultBucket   string            `toml:"default_bucket,omitempty"`
	PublicURLs      map[string]string `toml:"public_urls,omitempty"`
	PresignTTLSec   int               `toml:"presign_ttl_seconds,omitempty"`
}

// ClientFlags holds flag overrides parsed from argv.
type ClientFlags struct {
	Bucket string
	Prefix string
	Key    string
}

func clientConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "blobs", "config.toml"), nil
}

// LoadClientConfig merges (precedence: flags > env > .env-in-cwd > toml > defaults).
func LoadClientConfig(flags ClientFlags) (ClientConfig, error) {
	// .env (best-effort, matches server behavior)
	config.LoadDotEnv(".env")

	cfg := ClientConfig{}
	if path, err := clientConfigPath(); err == nil {
		if data, rerr := os.ReadFile(path); rerr == nil {
			_ = toml.Unmarshal(data, &cfg)
		} else if !os.IsNotExist(rerr) {
			return cfg, rerr
		}
	}

	// env overrides toml
	if v := os.Getenv("S3_ENDPOINT"); v != "" {
		cfg.Endpoint = v
	}
	if v := os.Getenv("R2_ACCOUNT_ID"); v != "" {
		cfg.R2AccountID = v
	}
	if cfg.Endpoint == "" && cfg.R2AccountID != "" {
		cfg.Endpoint = "https://" + cfg.R2AccountID + ".r2.cloudflarestorage.com"
	}
	if v := os.Getenv("S3_REGION"); v != "" {
		cfg.Region = v
	}
	if cfg.Region == "" {
		cfg.Region = "auto"
	}
	if v := config.Getenv("S3_ACCESS_KEY_ID", os.Getenv("R2_ACCESS_KEY_ID")); v != "" {
		cfg.AccessKeyID = v
	}
	if v := config.Getenv("S3_SECRET_ACCESS_KEY", os.Getenv("R2_SECRET_ACCESS_KEY")); v != "" {
		cfg.SecretAccessKey = v
	}
	if v := os.Getenv("BLOBS_DEFAULT_BUCKET"); v != "" {
		cfg.DefaultBucket = v
	}
	if v := os.Getenv("BLOBS_PUBLIC_URLS"); v != "" {
		merged := parsePublicURLs(v)
		if cfg.PublicURLs == nil {
			cfg.PublicURLs = map[string]string{}
		}
		for k, val := range merged {
			cfg.PublicURLs[k] = val
		}
	}
	if v := config.GetenvInt("BLOBS_PRESIGN_TTL_SECONDS", 0); v > 0 {
		cfg.PresignTTLSec = v
	}
	if cfg.PresignTTLSec <= 0 {
		cfg.PresignTTLSec = 3600
	}

	// flag overrides
	if flags.Bucket != "" {
		cfg.DefaultBucket = flags.Bucket
	}

	return cfg, nil
}

// SaveClientConfig writes cfg to ~/.config/blobs/config.toml (0600).
func SaveClientConfig(cfg ClientConfig) error {
	path, err := clientConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}

// NewS3FromConfig builds an S3 client from ClientConfig.
func NewS3FromConfig(cfg ClientConfig) (*S3Client, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("S3 endpoint not configured (set S3_ENDPOINT, R2_ACCOUNT_ID, or run `blobs auth`)")
	}
	pubs := cfg.PublicURLs
	if pubs == nil {
		pubs = map[string]string{}
	}
	ttl := time.Duration(cfg.PresignTTLSec) * time.Second
	return NewS3Client(cfg.Endpoint, cfg.Region, cfg.AccessKeyID, cfg.SecretAccessKey, pubs, ttl)
}

// ResolveURL returns a public URL if the bucket is mapped, else a presigned URL.
func ResolveURL(ctx context.Context, s3 *S3Client, bucket, key string) (string, error) {
	if u, ok := s3.PublicURL(bucket, key); ok {
		return u, nil
	}
	return s3.PresignGet(ctx, bucket, key)
}
