package tui

import (
	"path/filepath"
	"testing"
)

func writeTestConfig(t *testing.T, dir, remoteURL, apiKey string) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := SaveConfig(Config{RemoteURL: remoteURL, APIKey: apiKey}); err != nil {
		t.Fatalf("save config: %v", err)
	}
}

func TestResolveBackendUsesConfiguredRemoteByDefault(t *testing.T) {
	cfgDir := t.TempDir()
	writeTestConfig(t, cfgDir, "https://notes.example.com", "secret")
	t.Setenv("JOTTS_REMOTE_URL", "")
	t.Setenv("JOTTS_API_KEY", "")
	t.Setenv("JOTTS_DB_PATH", filepath.Join(t.TempDir(), "local.sqlite"))

	backend, err := ResolveBackend(Options{})
	if err != nil {
		t.Fatalf("resolve backend: %v", err)
	}
	defer backend.Close()

	remote, ok := backend.(*RemoteBackend)
	if !ok {
		t.Fatalf("expected remote backend, got %T", backend)
	}
	if remote.BaseURL != "https://notes.example.com" {
		t.Fatalf("expected configured remote URL, got %q", remote.BaseURL)
	}
	if remote.APIKey != "secret" {
		t.Fatalf("expected configured API key, got %q", remote.APIKey)
	}
}

func TestResolveBackendExplicitDBOverridesConfiguredRemote(t *testing.T) {
	cfgDir := t.TempDir()
	writeTestConfig(t, cfgDir, "https://notes.example.com", "secret")
	t.Setenv("JOTTS_REMOTE_URL", "")
	t.Setenv("JOTTS_API_KEY", "")

	dbPath := filepath.Join(t.TempDir(), "local.sqlite")
	backend, err := ResolveBackend(Options{DBPath: dbPath})
	if err != nil {
		t.Fatalf("resolve backend: %v", err)
	}
	defer backend.Close()

	if _, ok := backend.(*LocalBackend); !ok {
		t.Fatalf("expected local backend, got %T", backend)
	}
}

func TestResolveBackendExplicitRemoteOverridesExplicitDB(t *testing.T) {
	cfgDir := t.TempDir()
	writeTestConfig(t, cfgDir, "https://notes.example.com", "secret")
	t.Setenv("JOTTS_REMOTE_URL", "")
	t.Setenv("JOTTS_API_KEY", "")

	backend, err := ResolveBackend(Options{
		RemoteURL: "https://override.example.com",
		APIKey:    "override-key",
		DBPath:    filepath.Join(t.TempDir(), "local.sqlite"),
	})
	if err != nil {
		t.Fatalf("resolve backend: %v", err)
	}
	defer backend.Close()

	remote, ok := backend.(*RemoteBackend)
	if !ok {
		t.Fatalf("expected remote backend, got %T", backend)
	}
	if remote.BaseURL != "https://override.example.com" {
		t.Fatalf("expected explicit remote URL, got %q", remote.BaseURL)
	}
	if remote.APIKey != "override-key" {
		t.Fatalf("expected explicit API key, got %q", remote.APIKey)
	}
}
