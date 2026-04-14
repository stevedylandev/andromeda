use serde::{Deserialize, Serialize};
use std::path::{Path, PathBuf};

#[derive(Debug, Default, Serialize, Deserialize)]
pub struct Config {
    pub remote_url: Option<String>,
    pub api_key: Option<String>,
}

pub fn config_path() -> PathBuf {
    let home = std::env::var("HOME").unwrap_or_else(|_| ".".to_string());
    config_path_from(Path::new(&home))
}

pub fn config_path_from(home: &Path) -> PathBuf {
    home.join(".config/sipp/config.toml")
}

pub fn load_config() -> Config {
    load_config_from(&config_path())
}

pub fn load_config_from(path: &Path) -> Config {
    match std::fs::read_to_string(path) {
        Ok(contents) => toml::from_str(&contents).unwrap_or_default(),
        Err(_) => Config::default(),
    }
}

pub fn save_config(config: &Config) -> Result<(), Box<dyn std::error::Error>> {
    save_config_to(&config_path(), config)
}

pub fn save_config_to(path: &Path, config: &Config) -> Result<(), Box<dyn std::error::Error>> {
    if let Some(parent) = path.parent() {
        std::fs::create_dir_all(parent)?;
    }
    let contents = toml::to_string_pretty(config)?;
    std::fs::write(path, contents)?;
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn config_default_is_none_fields() {
        let config = Config::default();
        assert!(config.remote_url.is_none());
        assert!(config.api_key.is_none());
    }

    #[test]
    fn config_toml_roundtrip() {
        let config = Config {
            remote_url: Some("http://localhost:3000".to_string()),
            api_key: Some("secret-key-123".to_string()),
        };
        let serialized = toml::to_string_pretty(&config).unwrap();
        let deserialized: Config = toml::from_str(&serialized).unwrap();
        assert_eq!(deserialized.remote_url, config.remote_url);
        assert_eq!(deserialized.api_key, config.api_key);
    }

    #[test]
    fn config_toml_roundtrip_with_nones() {
        let config = Config {
            remote_url: None,
            api_key: None,
        };
        let serialized = toml::to_string_pretty(&config).unwrap();
        let deserialized: Config = toml::from_str(&serialized).unwrap();
        assert!(deserialized.remote_url.is_none());
        assert!(deserialized.api_key.is_none());
    }

    #[test]
    fn load_config_missing_file_returns_default() {
        let tmp = tempfile::tempdir().unwrap();
        let path = config_path_from(tmp.path());
        let config = load_config_from(&path);
        assert!(config.remote_url.is_none());
        assert!(config.api_key.is_none());
    }

    #[test]
    fn save_and_load_config_roundtrip() {
        let tmp = tempfile::tempdir().unwrap();
        let path = config_path_from(tmp.path());

        let config = Config {
            remote_url: Some("https://sipp.example.com".to_string()),
            api_key: Some("key123".to_string()),
        };
        save_config_to(&path, &config).unwrap();

        let loaded = load_config_from(&path);
        assert_eq!(loaded.remote_url, config.remote_url);
        assert_eq!(loaded.api_key, config.api_key);
    }
}
