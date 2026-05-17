package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestConfigTOMLRoundTrip(t *testing.T) {
	cfg := Config{RemoteURL: "https://example.test", APIKey: "secret"}
	var path = filepath.Join(t.TempDir(), "config.toml")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := toml.NewEncoder(f).Encode(cfg); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	var got Config
	if _, err := toml.DecodeFile(path, &got); err != nil {
		t.Fatal(err)
	}
	if got != cfg {
		t.Fatalf("got %#v want %#v", got, cfg)
	}
}

func TestLoadConfigMissingAndSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg != (Config{}) {
		t.Fatalf("missing config got %#v", cfg)
	}

	want := Config{RemoteURL: "http://localhost:3000", APIKey: "key"}
	if err := SaveConfig(want); err != nil {
		t.Fatal(err)
	}
	got, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %#v want %#v", got, want)
	}
}
