// Handles loading of main and site-specific YAML configurations.
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

// LogConfig holds logging settings from the main config.
type LogConfig struct {
	Level  string `yaml:"level"`
	Output string `yaml:"output"`
	Format string `yaml:"format"`
}

// MainConfig represents the contents of config.yaml.
type MainConfig struct {
	Port          int       `yaml:"port"`
	HTTPSCertPath string    `yaml:"https_cert_path"`
	HTTPSKeyPath  string    `yaml:"https_key_path"`
	Log           LogConfig `yaml:"log"`
}

// EndpointURL defines a backend URL and optional proxy/auth settings.
type EndpointURL struct {
	URL      string `yaml:"url"`
	Socks5   string `yaml:"socks5,omitempty"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
	Weight   int    `yaml:"weight,omitempty"`
}

// EndpointStrategy maps paths to strategies and URLs.
type EndpointStrategy map[string]struct {
	Strategy string        `yaml:"strategy"`
	URLs     []EndpointURL `yaml:"urls"`
}

// Exclusion defines temporary exclusion rules for backend responses.
type Exclusion struct {
	Type     string   `yaml:"type"`
	Match    []string `yaml:"match"`
	Duration int      `yaml:"duration"`
}

// EndpointConfig represents a site-specific configuration.
type EndpointConfig struct {
	Enable     bool             `yaml:"enable"`
	Endpoints  EndpointStrategy `yaml:"endpoints"`
	Exclusions []Exclusion      `yaml:"exclusions"`
}

// Reads config.yaml and unmarshals it into MainConfig.
func LoadMainConfig(path string) (*MainConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg MainConfig
	err = yaml.Unmarshal(data, &cfg)
	return &cfg, err
}

// Loads all YAML files (except config.yaml) with enable: true.
func LoadEnabledEndpointConfigs(dir string) ([]EndpointConfig, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var configs []EndpointConfig
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "config.yaml" || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}

		fullPath := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}

		var cfg EndpointConfig
		if err := yaml.Unmarshal(data, &cfg); err == nil && cfg.Enable {
			configs = append(configs, cfg)
		}
	}

	return configs, nil
}
