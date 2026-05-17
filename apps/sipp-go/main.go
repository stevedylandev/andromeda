// Sipp CLI: minimal command dispatcher.
//
//	sipp                              launch the interactive TUI
//	sipp tui [-r URL] [-k KEY]        launch the interactive TUI
//	sipp auth                         save remote URL + API key to config
//	sipp server [--host H] [--port P] start the web server
//	sipp [-r URL] [-k KEY] <file>     upload a file to a remote sipp server
//	sipp --help
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/stevedylandev/andromeda/apps/sipp-go/server"
	"github.com/stevedylandev/andromeda/apps/sipp-go/tui"
	"github.com/stevedylandev/andromeda/crates-go/config"
)

const usage = `sipp — minimal code sharing CLI

usage:
  sipp                              launch interactive TUI
  sipp tui [-r URL] [-k KEY]        launch interactive TUI
  sipp auth                         save remote URL + API key to ~/.config/sipp/config.toml
  sipp server [--host HOST] [--port PORT]
  sipp [-r URL] [-k KEY] <file>     create a snippet from FILE on the remote server
  sipp --help

env:
  SIPP_REMOTE_URL  default remote URL
  SIPP_API_KEY     API key used for authenticated requests
  SIPP_DB_PATH     local sqlite path for TUI in local mode
`

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		runTUI(nil)
		return
	}
	switch args[0] {
	case "-h", "--help":
		fmt.Print(usage)
	case "server":
		runServer(args[1:])
	case "tui":
		runTUI(args[1:])
	case "auth":
		runAuth()
	default:
		runUpload(args)
	}
}

func runServer(args []string) {
	config.LoadDotEnv(".env")
	host := config.Getenv("HOST", "127.0.0.1")
	port := config.GetenvInt("PORT", 3000)
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--host":
			if i+1 < len(args) {
				host = args[i+1]
				i++
			}
		case "--port", "-p":
			if i+1 < len(args) {
				if n, err := strconv.Atoi(args[i+1]); err == nil {
					port = n
				}
				i++
			}
		}
	}
	if err := server.Run(host, port); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runTUI(args []string) {
	if err := tui.Run(tui.ParseArgs(args)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runAuth() {
	cfg, _ := tui.LoadConfig()
	in := bufio.NewReader(os.Stdin)

	fmt.Printf("Remote URL [%s]: ", cfg.RemoteURL)
	url, _ := in.ReadString('\n')
	url = strings.TrimSpace(url)
	if url != "" {
		cfg.RemoteURL = url
	}

	masked := ""
	if cfg.APIKey != "" {
		masked = "********"
	}
	fmt.Printf("API key [%s]: ", masked)
	key, _ := in.ReadString('\n')
	key = strings.TrimSpace(key)
	if key != "" {
		cfg.APIKey = key
	}

	if err := tui.SaveConfig(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "save config:", err)
		os.Exit(1)
	}
	path, _ := tui.ConfigPath()
	fmt.Println("saved", path)
}

func runUpload(args []string) {
	remote := os.Getenv("SIPP_REMOTE_URL")
	apiKey := os.Getenv("SIPP_API_KEY")
	var file string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-r", "--remote":
			if i+1 < len(args) {
				remote = args[i+1]
				i++
			}
		case "-k", "--api-key":
			if i+1 < len(args) {
				apiKey = args[i+1]
				i++
			}
		default:
			if !strings.HasPrefix(args[i], "-") {
				file = args[i]
			}
		}
	}
	if file == "" {
		fmt.Fprintln(os.Stderr, "no file specified")
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	if remote == "" {
		cfg, _ := tui.LoadConfig()
		if cfg.RemoteURL != "" {
			remote = cfg.RemoteURL
		}
		if apiKey == "" {
			apiKey = cfg.APIKey
		}
	}
	if remote == "" {
		fmt.Fprintln(os.Stderr, "remote URL not set (use -r, SIPP_REMOTE_URL, or `sipp auth`)")
		os.Exit(2)
	}

	data, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	body, _ := json.Marshal(map[string]string{
		"name":    filepath.Base(file),
		"content": string(data),
	})
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(remote, "/")+"/api/snippets", bytes.NewReader(body))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("x-api-key", apiKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		fmt.Fprintf(os.Stderr, "server returned %s: %s\n", resp.Status, string(respBody))
		os.Exit(1)
	}
	var s server.Snippet
	if err := json.Unmarshal(respBody, &s); err != nil {
		fmt.Fprintln(os.Stderr, "could not parse response:", err)
		os.Exit(1)
	}
	fmt.Println(strings.TrimRight(remote, "/") + "/s/" + s.ShortID)
}
