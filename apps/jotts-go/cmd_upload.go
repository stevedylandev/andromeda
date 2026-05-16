package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/stevedylandev/andromeda/apps/jotts-go/tui"
)

func runUpload(args []string) {
	path := args[0]
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read file:", err)
		os.Exit(1)
	}
	title := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if title == "" {
		title = "Untitled"
	}

	backend, err := tui.ResolveBackend(tui.ParseArgs(args[1:]))
	if err != nil {
		fmt.Fprintln(os.Stderr, "backend:", err)
		os.Exit(1)
	}
	defer backend.Close()

	note, err := backend.Create(title, string(data))
	if err != nil {
		fmt.Fprintln(os.Stderr, "create note:", err)
		os.Exit(1)
	}

	if remote := backend.RemoteURL(); remote != "" {
		link := strings.TrimRight(remote, "/") + "/notes/" + note.ShortID
		fmt.Println(link)
		if err := clipboard.WriteAll(link); err == nil {
			fmt.Println("(copied to clipboard)")
		}
	} else {
		fmt.Println("created:", note.ShortID)
	}
}
