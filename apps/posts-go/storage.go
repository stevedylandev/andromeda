package main

import (
	"os"
	"path/filepath"
)

func readFileImpl(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}

func removeFile(path string) error {
	return os.Remove(path)
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func joinPath(parts ...string) string {
	return filepath.Join(parts...)
}
