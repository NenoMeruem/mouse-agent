package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	Engines  map[string]EngineConfig `mapstructure:"engines"`
	UI       UIConfig                `mapstructure:"ui"`
	Triggers TriggerConfig           `mapstructure:"triggers"`
}

type EngineConfig struct {
	APIKey string `mapstructure:"api_key"`
	Model  string `mapstructure:"model"`
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
			UI:      UIConfig{Output: "stdout"},
		}
		return nil
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return err
	}

	AppConfig = &cfg
	return nil
}
