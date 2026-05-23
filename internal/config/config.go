package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const appDirName = ".octocli_cg"
const configFileName = "config.yaml"

// Config is the top-level YAML configuration for octocli_cg.
type Config struct {
	DefaultProfile string             `yaml:"default_profile"`
	Profiles       map[string]Profile `yaml:"profiles"`
	Routing        RoutingConfig      `yaml:"routing"`
}

// Profile describes one OpenAI-compatible LLM endpoint.
type Profile struct {
	Name    string `yaml:"-"`
	BaseURL string `yaml:"base_url"`
	APIKey  string `yaml:"api_key"`
	Model   string `yaml:"model"`
}

// RoutingConfig reserves named model routes for fast/heavy task split.
type RoutingConfig struct {
	FastProfile  string `yaml:"fast_profile"`
	HeavyProfile string `yaml:"heavy_profile"`
}

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, appDirName, configFileName), nil
}

func Load(path string) (*Config, error) {
	if strings.TrimSpace(path) == "" {
		defaultPath, err := DefaultPath()
		if err != nil {
			return nil, err
		}
		path = defaultPath
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("config not found at %s; run `octocli_cg config init` first", path)
		}
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(contents, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func EnsureSample(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		defaultPath, err := DefaultPath()
		if err != nil {
			return "", err
		}
		path = defaultPath
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create config directory: %w", err)
	}
	if _, err := os.Stat(path); err == nil {
		return path, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("inspect config path: %w", err)
	}

	if err := os.WriteFile(path, []byte(SampleYAML()), 0o600); err != nil {
		return "", fmt.Errorf("write sample config: %w", err)
	}
	return path, nil
}

func (c *Config) Validate() error {
	if c == nil {
		return errors.New("config is nil")
	}
	if len(c.Profiles) == 0 {
		return errors.New("config must define at least one profile")
	}
	if strings.TrimSpace(c.DefaultProfile) == "" {
		return errors.New("config default_profile is required")
	}
	for name, profile := range c.Profiles {
		if strings.TrimSpace(profile.BaseURL) == "" {
			return fmt.Errorf("profile %q base_url is required", name)
		}
		if strings.TrimSpace(profile.Model) == "" {
			return fmt.Errorf("profile %q model is required", name)
		}
	}
	_, err := c.ResolveProfile(c.DefaultProfile)
	return err
}

func (c *Config) ResolveProfile(name string) (Profile, error) {
	if strings.TrimSpace(name) == "" {
		name = c.DefaultProfile
	}
	profile, ok := c.Profiles[name]
	if !ok {
		return Profile{}, fmt.Errorf("profile %q not found", name)
	}
	profile.Name = name
	return profile, nil
}

func SampleYAML() string {
	return `default_profile: local

profiles:
  local:
    base_url: http://localhost:11434/v1
    api_key: ""
    model: llama3.1
  openai:
    base_url: https://api.openai.com/v1
    api_key: ${OPENAI_API_KEY}
    model: gpt-4o-mini
  custom:
    base_url: https://example.com/v1
    api_key: ${CUSTOM_API_KEY}
    model: custom-model

routing:
  fast_profile: local
  heavy_profile: openai
`
}
