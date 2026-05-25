package main

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
)

type tuiOptions struct {
	Bucket string
	Prefix string
}

func parseTUIArgs(args []string) tuiOptions {
	var opts tuiOptions
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case (a == "-b" || a == "--bucket") && i+1 < len(args):
			opts.Bucket = args[i+1]
			i++
		case strings.HasPrefix(a, "--bucket="):
			opts.Bucket = strings.TrimPrefix(a, "--bucket=")
		case a == "--prefix" && i+1 < len(args):
			opts.Prefix = args[i+1]
			i++
		case strings.HasPrefix(a, "--prefix="):
			opts.Prefix = strings.TrimPrefix(a, "--prefix=")
		}
	}
	return opts
}

func runTUI(args []string) {
	opts := parseTUIArgs(args)
	cfg, err := LoadClientConfig(ClientFlags{Bucket: opts.Bucket})
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}
	s3, err := NewS3FromConfig(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "s3:", err)
		os.Exit(1)
	}

	m := newTUIModel(s3, cfg, opts)
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
