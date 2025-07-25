// Handles loading of main and site-specific YAML configurations.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

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

// Validate checks for required fields in MainConfig.
func (c *MainConfig) Validate() error {
	if c.Port == 0 {
		return fmt.Errorf("port must be specified and non-zero")
	}
	if c.Log.Level == "" {
		return fmt.Errorf("log.level must be specified")
	}
	if c.Log.Output == "" {
		return fmt.Errorf("log.output must be specified")
	}
	if c.Log.Format == "" {
		return fmt.Errorf("log.format must be specified")
	}
	return nil
}

// Reads config.yaml and unmarshals it into MainConfig.
func LoadMainConfig(path string) (*MainConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg MainConfig
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}
	err = cfg.Validate()
	if err != nil {
		return nil, err
	}
	return &cfg, nil
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

		// find duplicate paths within same file
		if dup := findDuplicateEndpointWithinFile(data); dup != "" {
			return nil, fmt.Errorf("duplicate endpoint path found: %s in %s", dup, fullPath)
		}

		var cfg EndpointConfig
		if err := yaml.Unmarshal(data, &cfg); err == nil && cfg.Enable {
			configs = append(configs, cfg)
		}
	}

	// find duplicate paths across files
	if err = findDuplicateEndpointAcrossFiles(configs); err != nil {
		return nil, err
	}

	return configs, nil
}

// Find duplicate endpoint within same file, returns duplicate endpoint path
func findDuplicateEndpointWithinFile(data []byte) string {
	text := string(data)

	// Remove comments from # to end of line
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if idx := strings.Index(line, "#"); idx != -1 {
			lines[i] = line[:idx]
		}
	}
	cleaned := strings.Join(lines, "\n")

	// Find all occurrences of "/word":
	re := regexp.MustCompile(`"(/[\w\d]+)":`)
	matches := re.FindAllStringSubmatch(cleaned, -1)

	seen := make(map[string]bool)
	for _, match := range matches {
		word := match[0] // full match like "/abc":
		if seen[word] {
			return word
		}
		seen[word] = true
	}

	return ""
}

// Find duplicate endpoint across files
func findDuplicateEndpointAcrossFiles(configs []EndpointConfig) error {
	seen := make(map[string]string) // path -> filename or index

	for i, cfg := range configs {
		for path := range cfg.Endpoints {
			if prev, exists := seen[path]; exists {
				return fmt.Errorf("duplicate path %q found in config[%d], also seen in %s", path, i, prev)
			}
			seen[path] = fmt.Sprintf("config[%d]", i)
		}
	}

	return nil
}
