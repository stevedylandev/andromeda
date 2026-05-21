package tui

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config is the on-disk TUI config shape shared across apps.
type Config struct {
	RemoteURL string `toml:"remote_url"`
	APIKey    string `toml:"api_key"`
}

// ConfigPath returns $XDG_CONFIG_HOME/<app>/config.toml.
func ConfigPath(app string) (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, app, "config.toml"), nil
}

// LoadConfig reads the named app's config. Missing file returns a zero Config.
func LoadConfig(app string) (Config, error) {
	var cfg Config
	path, err := ConfigPath(app)
	if err != nil {
		return cfg, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// SaveConfig writes cfg as TOML, creating parent dirs as needed.
func SaveConfig(app string, cfg Config) error {
	path, err := ConfigPath(app)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}
