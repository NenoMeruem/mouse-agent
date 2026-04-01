// Package config handles loading and managing application configuration.
// Configuration can come from YAML files or environment variables.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Default configuration values - single source of truth for defaults
const (
	DefaultOpenAIModel       = "gpt-4-mini"
	DefaultGeminiModel       = "gemini-2.5-flash-lite"
	DefaultClaudeModel       = "claude-sonnet-4-6"
	DefaultTimeout           = 120 * time.Second // Timeout for API requests
	DefaultUIOutput          = "stdout"
	DefaultSelectionProvider = "auto"
)

type Config struct {
	Engines   map[string]EngineConfig `mapstructure:"engines"`
	UI        UIConfig                `mapstructure:"ui"`
	Triggers  TriggerConfig           `mapstructure:"triggers"`
	Selection SelectionConfig         `mapstructure:"selection"`
}

type EngineConfig struct {
	APIKey  string        `mapstructure:"api_key"`
	Model   string        `mapstructure:"model"`
	Timeout time.Duration `mapstructure:"timeout"`
}

type UIConfig struct {
	Output string `mapstructure:"output"`
}

type TriggerConfig struct {
	Enabled       bool                   `mapstructure:"enabled"`
	DefaultPrompt string                 `mapstructure:"default_prompt"`
	Hotkeys       map[string]string      `mapstructure:"hotkeys"` // hotkey -> prompt_id
	Mouse         map[string]interface{} `mapstructure:"mouse"`
}

type SelectionConfig struct {
	Provider             string `mapstructure:"provider"` // auto, macos, linux, windows
	FailOnEmptySelection bool   `mapstructure:"fail_on_empty_selection"`
	TrimWhitespace       bool   `mapstructure:"trim_whitespace"`
}

var AppConfig *Config

func Load() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot get home directory: %w", err)
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(filepath.Join(home, ".prompt-agent"))

	viper.SetEnvPrefix("PROMPT_AGENT")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		// Ignore error if config file doesn't exist, just use defaults
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("cannot read config: %w", err)
		}
		AppConfig = &Config{
			Engines: make(map[string]EngineConfig),
			UI:      UIConfig{Output: DefaultUIOutput},
		}
		return nil
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return err
	}

	// Ensure all engines have valid timeout
	for engine, engineCfg := range cfg.Engines {
		if engineCfg.Timeout <= 0 {
			engineCfg.Timeout = DefaultTimeout
		}
		cfg.Engines[engine] = engineCfg
	}

	// Apply defaults for UI output if not specified
	if cfg.UI.Output == "" {
		cfg.UI.Output = DefaultUIOutput
	}

	AppConfig = &cfg
	return nil
}

// GetEngineModel returns the model for the given engine, falling back to defaults.
func GetEngineModel(engine string) string {
	if AppConfig != nil && AppConfig.Engines != nil {
		if cfg, ok := AppConfig.Engines[engine]; ok && cfg.Model != "" {
			return cfg.Model
		}
	}

	// Return engine-specific defaults
	switch engine {
	case "openai":
		return DefaultOpenAIModel
	case "gemini":
		return DefaultGeminiModel
	case "claude":
		return DefaultClaudeModel
	default:
		return DefaultOpenAIModel
	}
}

// GetEngineTimeout returns the timeout for the given engine, falling back to default.
func GetEngineTimeout(engine string) time.Duration {
	if AppConfig != nil && AppConfig.Engines != nil {
		if cfg, ok := AppConfig.Engines[engine]; ok && cfg.Timeout > 0 {
			return cfg.Timeout
		}
	}
	return DefaultTimeout
}

// GetUIOutput returns the UI output type, falling back to default.
func GetUIOutput() string {
	if AppConfig != nil && AppConfig.UI.Output != "" {
		return AppConfig.UI.Output
	}
	return DefaultUIOutput
}

// resolveAPIKey resolves an "env:VARNAME" prefix to the actual environment variable value.
// If the raw string doesn't start with "env:", it is returned as-is.
func resolveAPIKey(raw string) string {
	if strings.HasPrefix(raw, "env:") {
		return os.Getenv(strings.TrimPrefix(raw, "env:"))
	}
	return raw
}

// GetEngineAPIKey returns the resolved API key for the given engine.
// It handles the "env:VARNAME" prefix by reading the named environment variable.
func GetEngineAPIKey(engine string) string {
	if AppConfig != nil && AppConfig.Engines != nil {
		if cfg, ok := AppConfig.Engines[engine]; ok && cfg.APIKey != "" {
			return resolveAPIKey(cfg.APIKey)
		}
	}
	return ""
}
