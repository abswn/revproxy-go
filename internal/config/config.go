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

// URLConfig defines a single backend URL and optional proxy/auth settings.
type URLConfig struct {
	URL      string `yaml:"url"`
	Socks5   string `yaml:"socks5,omitempty"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
	Weight   int    `yaml:"weight,omitempty"`
}

// BanRule defines the matching words and the duration of ban for backend URLs.
type BanRule struct {
	Match    []string `yaml:"match"`
	Duration int      `yaml:"duration"`
}

// StrategyConfig defines a stategy, a slice of backend URLs to use for the strategy and the ban rules.
type StrategyConfig struct {
	Strategy string      `yaml:"strategy"`
	URLs     []URLConfig `yaml:"urls"`
	BanRules []BanRule   `yaml:"ban,omitempty"`
}

// EndpointsConfig represents all endpoints in a config file keyed by their paths.
type EndpointsConfig struct {
	Enabled        bool                      `yaml:"enabled"`
	EndpointsMap   map[string]StrategyConfig `yaml:"endpoints"`
	GlobalBanRules []BanRule                 `yaml:"global_ban"`
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

// Replace empty local ban rules with global ban rules
func applyGlobalBanRules(config *EndpointsConfig) {
	// iterate over endpoint path and strategyConfig pairs
	for path, strat := range config.EndpointsMap {
		if len(strat.BanRules) == 0 {
			strat.BanRules = config.GlobalBanRules
			config.EndpointsMap[path] = strat
		}
	}
}

// Loads all YAML files (except config.yaml) with enabled: true.
func LoadEnabledEndpointsMap(dir string) (map[string]StrategyConfig, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	configs := make(map[string]StrategyConfig)
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

		var cfg EndpointsConfig
		if err := yaml.Unmarshal(data, &cfg); err == nil && cfg.Enabled {
			applyGlobalBanRules(&cfg)
			for path, strat := range cfg.EndpointsMap {
				if _, exists := configs[path]; exists {
					return nil, fmt.Errorf("duplicate endpoint path found: %s", path)
				}
				configs[path] = strat
			}
		} else if err != nil {
			return nil, fmt.Errorf("failed to parse endpoint config %s: %v", entry.Name(), err)
		}
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
