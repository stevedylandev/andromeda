package tui

import "github.com/pkg/browser"

func openURL(url string) error {
	return browser.OpenURL(url)
}
