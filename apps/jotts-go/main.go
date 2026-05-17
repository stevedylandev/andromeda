package main

import (
	"fmt"
	"os"

	"github.com/stevedylandev/andromeda/apps/jotts-go/tui"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		runTUI(nil)
		return
	}

	switch args[0] {
	case "server":
		runServer(args[1:])
	case "tui":
		runTUI(args[1:])
	case "auth":
		runAuth(args[1:])
	case "-h", "--help", "help":
		printUsage()
	default:
		if _, err := os.Stat(args[0]); err == nil {
			runUpload(args)
			return
		}
		runTUI(args)
	}
}

func runTUI(args []string) {
	if err := tui.Run(tui.ParseArgs(args)); err != nil {
		fmt.Fprintln(os.Stderr, "tui error:", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`jotts-go — minimal markdown notes

usage:
  jotts-go                            launch TUI (default)
  jotts-go tui  [--remote URL --api-key KEY]
  jotts-go server                     run HTTP server
  jotts-go auth                       configure remote URL + API key
  jotts-go <file.md>                  upload file as a new note`)
}
