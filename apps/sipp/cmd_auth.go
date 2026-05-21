package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/stevedylandev/andromeda/apps/sipp/tui"
	"golang.org/x/term"
)

func runAuth(_ []string) {
	cfg, _ := tui.LoadConfig()
	reader := bufio.NewReader(os.Stdin)

	defaultURL := cfg.RemoteURL
	if defaultURL == "" {
		defaultURL = "http://localhost:3000"
	}
	fmt.Printf("Remote URL [%s]: ", defaultURL)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line != "" {
		cfg.RemoteURL = line
	} else {
		cfg.RemoteURL = defaultURL
	}

	fmt.Print("API key (hidden): ")
	keyBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		fmt.Fprintln(os.Stderr, "read api key:", err)
		os.Exit(1)
	}
	if k := strings.TrimSpace(string(keyBytes)); k != "" {
		cfg.APIKey = k
	}

	if err := tui.SaveConfig(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "save config:", err)
		os.Exit(1)
	}
	path, _ := tui.ConfigPath()
	fmt.Println("Saved", path)
}
