package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/stevedylandev/andromeda/apps/sipp/tui"
)

func runUpload(args []string) {
	var file string
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			file = a
			break
		}
	}
	if file == "" {
		fmt.Fprintln(os.Stderr, "no file specified")
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	data, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read file:", err)
		os.Exit(1)
	}
	name := filepath.Base(file)

	opts := tui.ParseArgs(args)
	if opts.RemoteURL == "" {
		opts.RemoteURL = os.Getenv("SIPP_REMOTE_URL")
	}
	if opts.RemoteURL == "" {
		cfg, _ := tui.LoadConfig()
		opts.RemoteURL = cfg.RemoteURL
	}
	if opts.RemoteURL == "" {
		fmt.Fprintln(os.Stderr, "remote URL not set (use -r, SIPP_REMOTE_URL, or `sipp auth`)")
		os.Exit(2)
	}

	backend, err := tui.ResolveBackend(opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "backend:", err)
		os.Exit(1)
	}
	defer backend.Close()

	snippet, err := backend.Create(name, string(data))
	if err != nil {
		fmt.Fprintln(os.Stderr, "create snippet:", err)
		os.Exit(1)
	}

	if remote := backend.RemoteURL(); remote != "" {
		link := strings.TrimRight(remote, "/") + "/s/" + snippet.ShortID
		fmt.Println(link)
		if err := clipboard.WriteAll(link); err == nil {
			fmt.Println("(copied to clipboard)")
		}
	} else {
		fmt.Println("created:", snippet.ShortID)
	}
}
