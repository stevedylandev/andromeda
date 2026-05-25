// blobs — S3 browser, CLI uploader, and TUI.
//
//	blobs                              launch the interactive TUI
//	blobs tui [-b BUCKET]              launch the TUI (alias)
//	blobs auth                         interactive client config writer
//	blobs server [--host H] [--port P] run the web server
//	blobs [-b BUCKET] <file>           upload a file and print its URL
//	blobs --help
package main

import (
	"fmt"
	"os"
)

const usage = `blobs — S3 browser, CLI, and TUI

usage:
  blobs                              launch interactive TUI
  blobs tui [-b BUCKET] [--prefix P] launch TUI
  blobs auth                         write client config to ~/.config/blobs/config.toml
  blobs server [--host H] [--port P] run the web server
  blobs [-b BUCKET] [--prefix P] [--key K] <file>
                                     upload FILE, print URL to stdout
  blobs --help

env:
  BLOBS_DEFAULT_BUCKET   bucket used when -b is omitted
  S3_ENDPOINT            S3 endpoint URL
  S3_REGION              region (default "auto")
  S3_ACCESS_KEY_ID       access key
  S3_SECRET_ACCESS_KEY   secret key
  R2_ACCOUNT_ID          shortcut: derives endpoint for Cloudflare R2
  BLOBS_PUBLIC_URLS      bucket=url,bucket=url public URL map
  BLOBS_PRESIGN_TTL_SECONDS  presigned URL lifetime (default 3600)
  BLOBS_PREVIEW          override preview backend: kitty|iterm|chafa|none
`

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		runTUI(nil)
		return
	}
	switch args[0] {
	case "-h", "--help", "help":
		fmt.Print(usage)
	case "server":
		runServer(args[1:])
	case "tui":
		runTUI(args[1:])
	case "auth":
		runAuth(args[1:])
	default:
		if _, err := os.Stat(args[0]); err == nil {
			runUpload(args)
			return
		}
		// no file at args[0] — treat as TUI flags
		runTUI(args)
	}
}
