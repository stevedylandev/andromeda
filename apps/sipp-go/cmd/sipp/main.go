// Sipp CLI: minimal command dispatcher.
//
//	sipp server              start the web server
//	sipp [-r URL] [-k KEY] <file>   upload a file to a remote sipp server
//	sipp --help
//
// The interactive TUI from the Rust version is not ported.
package main

import (
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
	"github.com/stevedylandev/andromeda/crates-go/config"
)

const usage = `sipp — minimal code sharing CLI

usage:
  sipp server [--host HOST] [--port PORT]
  sipp [-r URL] [-k KEY] <file>     create a snippet from FILE on the remote server
  sipp --help

env:
  SIPP_REMOTE_URL  default remote URL
  SIPP_API_KEY     API key used for authenticated requests
`

func main() {
	config.LoadDotEnv(".env")
	args := os.Args[1:]
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		fmt.Print(usage)
		return
	}
	switch args[0] {
	case "server":
		runServer(args[1:])
	default:
		runUpload(args)
	}
}

func runServer(args []string) {
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
		fmt.Fprintln(os.Stderr, "remote URL not set (use -r or SIPP_REMOTE_URL)")
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
