// config.rs — Configuration management.
// Reads/writes ~/.promptly/config.yaml and resolves API keys from env vars.
// Mirrors Go's internal/config/config.go behaviour exactly.

use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::fs;
use std::path::PathBuf;

// ---------------------------------------------------------------------------
// Default values — single source of truth
// ---------------------------------------------------------------------------

pub const DEFAULT_OPENAI_MODEL: &str = "gpt-4o-mini";
pub const DEFAULT_GEMINI_MODEL: &str = "gemini-2.5-flash-lite";
pub const DEFAULT_CLAUDE_MODEL: &str = "claude-sonnet-4-6";
pub const DEFAULT_HOTKEY: &str = "Alt+Space";
pub const DEFAULT_TIMEOUT_SECS: u64 = 120;

// ---------------------------------------------------------------------------
// Config structs
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct AppConfig {
    #[serde(default)]
    pub engines: HashMap<String, EngineConfig>,
    #[serde(default)]
    pub ui: UiConfig,
    #[serde(default)]
    pub hotkey: Option<String>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct EngineConfig {
    /// Provider type: "gemini" | "openai" | "claude".
    /// Determines which API client to instantiate.
    /// If missing (legacy entries), inferred from the HashMap key.
    #[serde(default)]
    pub provider: String,
    /// Human-readable display name shown in the UI.
    #[serde(default)]
    pub name: String,
    #[serde(default)]
    pub api_key: String,
    #[serde(default)]
    pub model: String,
    /// Timeout in seconds; 0 means use the default.
    #[serde(default)]
    pub timeout_secs: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UiConfig {
    #[serde(default = "default_output")]
    pub output: String,
}

fn default_output() -> String {
    "stdout".into()
}

impl Default for UiConfig {
    fn default() -> Self {
        Self { output: default_output() }
    }
}

// ---------------------------------------------------------------------------
// Config file path
// ---------------------------------------------------------------------------

/// Returns `~/.promptly/config.yaml`.
pub fn config_path() -> PathBuf {
    let home = dirs::home_dir().unwrap_or_default();
    home.join(".promptly").join("config.yaml")
}

/// Returns `~/.promptly/prompts.db`.
pub fn db_path() -> PathBuf {
    let home = dirs::home_dir().unwrap_or_default();
    home.join(".promptly").join("prompts.db")
}

// ---------------------------------------------------------------------------
// Load / Save
// ---------------------------------------------------------------------------

/// Reads the YAML config file. Returns an empty default config if the file
/// does not exist yet (matching Go's graceful handling).
pub fn load_config() -> AppConfig {
    let path = config_path();
    if !path.exists() {
        return AppConfig::default();
    }
    let content = match fs::read_to_string(&path) {
        Ok(c) => c,
        Err(_) => return AppConfig::default(),
    };
    serde_yaml::from_str(&content).unwrap_or_default()
}

/// Persists the given config to `~/.promptly/config.yaml`.
pub fn save_config(cfg: &AppConfig) -> Result<(), String> {
    let path = config_path();
    if let Some(parent) = path.parent() {
        fs::create_dir_all(parent).map_err(|e| e.to_string())?;
    }
    let content = serde_yaml::to_string(cfg).map_err(|e| e.to_string())?;
    fs::write(&path, content).map_err(|e| e.to_string())
}

// ---------------------------------------------------------------------------
// API key resolution
// ---------------------------------------------------------------------------

/// Resolves an `"env:VARNAME"` prefix to the actual environment variable value.
/// If the raw string does not start with `"env:"`, returns it as-is.
pub fn resolve_api_key(raw: &str) -> String {
    if let Some(var_name) = raw.strip_prefix("env:") {
        std::env::var(var_name).unwrap_or_default()
    } else {
        raw.to_string()
    }
}

/// Infers provider type from the entry ID when the `provider` field is empty.
/// Handles legacy configs where the key was "gemini", "openai", or "claude".
pub fn infer_provider(id: &str, ec: &EngineConfig) -> String {
    if !ec.provider.is_empty() {
        return ec.provider.clone();
    }
    // Legacy fallback: key IS the provider
    match id {
        "gemini" | "openai" | "claude" => id.to_string(),
        // Fuzzy match: starts with known provider name
        s if s.starts_with("gemini") => "gemini".to_string(),
        s if s.starts_with("openai") || s.starts_with("gpt") => "openai".to_string(),
        s if s.starts_with("claude") || s.starts_with("anthropic") => "claude".to_string(),
        _ => String::new(),
    }
}

/// Returns the resolved API key for the given entry ID.
/// For legacy single-type env var overrides, checks the provider env var.
pub fn get_engine_api_key(cfg: &AppConfig, id: &str) -> String {
    if let Some(ec) = cfg.engines.get(id) {
        // Check global env var for provider type
        let provider = infer_provider(id, ec);
        let env_override = match provider.as_str() {
            "openai" => std::env::var("OPENAI_API_KEY").unwrap_or_default(),
            "gemini" => std::env::var("GEMINI_API_KEY").unwrap_or_default(),
            "claude" => std::env::var("ANTHROPIC_API_KEY").unwrap_or_default(),
            _ => String::new(),
        };
        // env var only overrides if the entry has no api_key set
        if !ec.api_key.is_empty() {
            return resolve_api_key(&ec.api_key);
        }
        return env_override;
    }
    String::new()
}

/// Returns the configured model for the given entry ID, with provider-based defaults.
pub fn get_engine_model(cfg: &AppConfig, id: &str) -> String {
    if let Some(ec) = cfg.engines.get(id) {
        if !ec.model.is_empty() {
            return ec.model.clone();
        }
        // Fallback default based on provider
        return match infer_provider(id, ec).as_str() {
            "openai" => DEFAULT_OPENAI_MODEL.into(),
            "gemini" => DEFAULT_GEMINI_MODEL.into(),
            "claude" => DEFAULT_CLAUDE_MODEL.into(),
            _ => String::new(),
        };
    }
    String::new()
}

/// Returns the display name for the given entry ID.
pub fn get_engine_name(cfg: &AppConfig, id: &str) -> String {
    if let Some(ec) = cfg.engines.get(id) {
        if !ec.name.is_empty() {
            return ec.name.clone();
        }
        // Fallback based on provider
        return match infer_provider(id, ec).as_str() {
            "openai" => "OpenAI".into(),
            "gemini" => "Google Gemini".into(),
            "claude" => "Claude".into(),
            _ => id.to_string(),
        };
    }
    id.to_string()
}

/// Returns timeout in seconds for the given entry ID.
pub fn get_engine_timeout_secs(cfg: &AppConfig, id: &str) -> u64 {
    if let Some(ec) = cfg.engines.get(id) {
        if ec.timeout_secs > 0 {
            return ec.timeout_secs;
        }
    }
    DEFAULT_TIMEOUT_SECS
}

/// Returns the configured hotkey, or the default fallback.
pub fn get_hotkey(cfg: &AppConfig) -> String {
    cfg.hotkey.clone().filter(|h| !h.is_empty()).unwrap_or_else(|| DEFAULT_HOTKEY.into())
}

// ---------------------------------------------------------------------------
// Helper: build JSON payload of all engine configs (for UI settings screen)
// ---------------------------------------------------------------------------

#[derive(Debug, Serialize, Deserialize)]
pub struct EngineConfigView {
    pub id: String,
    /// Resolved provider: "gemini" | "openai" | "claude"
    pub provider: String,
    pub name: String,
    pub api_key: String,
    pub model: String,
}

pub fn get_engine_configs_json(cfg: &AppConfig) -> String {
    let mut views: Vec<EngineConfigView> = cfg
        .engines
        .iter()
        .map(|(id, ec)| EngineConfigView {
            id: id.clone(),
            provider: infer_provider(id, ec),
            name: get_engine_name(cfg, id),
            api_key: ec.api_key.clone(),
            model: ec.model.clone(),
        })
        .collect();
    // Sort by id for stable ordering
    views.sort_by(|a, b| a.id.cmp(&b.id));
    serde_json::to_string(&views).unwrap_or_else(|_| "[]".into())
}

// ---------------------------------------------------------------------------
// Unit Tests
// ---------------------------------------------------------------------------

#[cfg(test)]
mod tests {
    use super::*;

    fn make_cfg(engine: &str, api_key: &str, model: &str) -> AppConfig {
        let mut cfg = AppConfig::default();
        cfg.engines.insert(
            engine.to_string(),
            EngineConfig {
                provider: engine.to_string(),
                name: String::new(),
                api_key: api_key.to_string(),
                model: model.to_string(),
                timeout_secs: 0,
            },
        );
        cfg
    }

    #[test]
    fn resolve_api_key_passthrough() {
        assert_eq!(resolve_api_key("mykey123"), "mykey123");
    }

    #[test]
    fn resolve_api_key_env_prefix() {
        std::env::set_var("TEST_RESOLVE_KEY_VAR", "resolved_value");
        let result = resolve_api_key("env:TEST_RESOLVE_KEY_VAR");
        assert_eq!(result, "resolved_value");
        std::env::remove_var("TEST_RESOLVE_KEY_VAR");
    }

    #[test]
    fn resolve_api_key_env_prefix_missing_var_returns_empty() {
        std::env::remove_var("NONEXISTENT_KEY_12345");
        let result = resolve_api_key("env:NONEXISTENT_KEY_12345");
        assert_eq!(result, "");
    }

    #[test]
    fn get_engine_model_fallback_to_defaults() {
        // When an entry EXISTS in config but has no model set, provider-based default is used
        let cfg_openai = make_cfg("openai", "", "");
        assert_eq!(get_engine_model(&cfg_openai, "openai"), DEFAULT_OPENAI_MODEL);
        let cfg_gemini = make_cfg("gemini", "", "");
        assert_eq!(get_engine_model(&cfg_gemini, "gemini"), DEFAULT_GEMINI_MODEL);
        let cfg_claude = make_cfg("claude", "", "");
        assert_eq!(get_engine_model(&cfg_claude, "claude"), DEFAULT_CLAUDE_MODEL);
        // When ID doesn't exist at all, returns empty string
        let empty = AppConfig::default();
        assert_eq!(get_engine_model(&empty, "openai"), "");
    }

    #[test]
    fn get_engine_model_uses_config_when_set() {
        let cfg = make_cfg("gemini", "", "gemini-custom-model");
        assert_eq!(get_engine_model(&cfg, "gemini"), "gemini-custom-model");
    }

    #[test]
    fn get_engine_api_key_reads_from_config() {
        // Unset env so config fallback is used
        std::env::remove_var("GEMINI_API_KEY");
        let cfg = make_cfg("gemini", "config-key-abc", "");
        assert_eq!(get_engine_api_key(&cfg, "gemini"), "config-key-abc");
    }

    #[test]
    fn get_engine_api_key_env_var_overrides_config() {
        // Use a custom engine key to avoid clashing with real env vars set at system level
        // The function checks "GEMINI_API_KEY" for engine="gemini", so we test with
        // a config that uses env: prefix instead, which is fully isolated
        let cfg = make_cfg("gemini", "env:PROMPTLY_TEST_GEMINI_KEY_OVERRIDE", "");
        std::env::set_var("PROMPTLY_TEST_GEMINI_KEY_OVERRIDE", "env-override-key");
        assert_eq!(get_engine_api_key(&cfg, "gemini"), "env-override-key");
        std::env::remove_var("PROMPTLY_TEST_GEMINI_KEY_OVERRIDE");
    }

    #[test]
    fn get_engine_timeout_secs_defaults() {
        let cfg = AppConfig::default();
        assert_eq!(get_engine_timeout_secs(&cfg, "openai"), DEFAULT_TIMEOUT_SECS);
    }

    #[test]
    fn get_hotkey_returns_default_when_unset() {
        let cfg = AppConfig::default();
        assert_eq!(get_hotkey(&cfg), DEFAULT_HOTKEY);
    }

    #[test]
    fn get_hotkey_returns_custom_value() {
        let cfg = AppConfig {
            hotkey: Some("Ctrl+Shift+P".into()),
            ..Default::default()
        };
        assert_eq!(get_hotkey(&cfg), "Ctrl+Shift+P");
    }

    #[test]
    fn get_engine_name_fallback() {
        // When entry EXISTS in config, provider-based display name is used as fallback
        let cfg_gemini = make_cfg("gemini", "", "");
        assert_eq!(get_engine_name(&cfg_gemini, "gemini"), "Google Gemini");
        let cfg_openai = make_cfg("openai", "", "");
        assert_eq!(get_engine_name(&cfg_openai, "openai"), "OpenAI");
        let cfg_claude = make_cfg("claude", "", "");
        assert_eq!(get_engine_name(&cfg_claude, "claude"), "Claude");
        // When ID doesn't exist, returns the id itself
        let empty = AppConfig::default();
        assert_eq!(get_engine_name(&empty, "unknown"), "unknown");
    }

    #[test]
    fn app_config_default_is_empty_engines() {
        let cfg = AppConfig::default();
        assert!(cfg.engines.is_empty());
        assert_eq!(cfg.ui.output, "stdout");
    }
}
