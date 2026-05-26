package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"golang.org/x/term"
)

func runAuth(_ []string) {
	cfg, _ := LoadClientConfig(ClientFlags{})
	reader := bufio.NewReader(os.Stdin)

	cfg.Endpoint = promptDefault(reader, "S3 endpoint (or 'r2' for Cloudflare R2)", cfg.Endpoint)
	if strings.EqualFold(cfg.Endpoint, "r2") {
		cfg.R2AccountID = promptDefault(reader, "R2 account ID", cfg.R2AccountID)
		cfg.Endpoint = "https://" + cfg.R2AccountID + ".r2.cloudflarestorage.com"
	}

	region := cfg.Region
	if region == "" {
		region = "auto"
	}
	cfg.Region = promptDefault(reader, "Region", region)

	cfg.AccessKeyID = promptDefault(reader, "Access key ID", cfg.AccessKeyID)

	fmt.Print("Secret access key (hidden): ")
	secretBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		fmt.Fprintln(os.Stderr, "read secret:", err)
		os.Exit(1)
	}
	if s := strings.TrimSpace(string(secretBytes)); s != "" {
		cfg.SecretAccessKey = s
	}

	cfg.DefaultBucket = promptDefault(reader, "Default bucket (optional)", cfg.DefaultBucket)

	if cfg.DefaultBucket != "" {
		existing := ""
		if cfg.PublicURLs != nil {
			existing = cfg.PublicURLs[cfg.DefaultBucket]
		}
		pub := promptDefault(reader, "Public URL for "+cfg.DefaultBucket+" (optional)", existing)
		if pub != "" {
			if cfg.PublicURLs == nil {
				cfg.PublicURLs = map[string]string{}
			}
			cfg.PublicURLs[cfg.DefaultBucket] = pub
		}
	}

	if cfg.PresignTTLSec <= 0 {
		cfg.PresignTTLSec = 3600
	}

	if err := SaveClientConfig(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "save:", err)
		os.Exit(1)
	}
	path, _ := clientConfigPath()
	fmt.Println("Saved", path)
}

func promptDefault(r *bufio.Reader, label, def string) string {
	if def != "" {
		fmt.Printf("%s [%s]: ", label, def)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}
