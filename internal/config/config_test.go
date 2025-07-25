package config

import (
	"os"
	"path/filepath"
	"testing"
)

const mainConfigYAML = `
port: 8080
https_cert_path: ""
https_key_path: ""
log:
  level: "info"
  output: "stdout"
  format: "text"
`

const validEndpointConfigYAML = `
enable: true
endpoints:
  "/path1":
    strategy: round-robin
    urls:
      - url: "https://example.com/api1"
  "/path2":
    strategy: weighted
    urls:
      - url: "https://example.com/api2"
        weight: 50
exclusions:
  - type: "short"
    match: ["error"]
    duration: 60
`

const duplicateEndpointConfigYAML = `
enable: true
endpoints:
  "/path1":
    strategy: round-robin
    urls:
      - url: "https://example.com/api1"
  "/path1":
    strategy: weighted
    urls:
      - url: "https://example.com/api2"
        weight: 50
`

func TestLoadMainConfig_Valid(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "config.yaml")
	if err := os.WriteFile(configPath, []byte(mainConfigYAML), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadMainConfig(configPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Port)
	}
}

func TestLoadEnabledEndpointConfigs_Valid(t *testing.T) {
	tmp := t.TempDir()

	// Write main config (ignored by loader)
	_ = os.WriteFile(filepath.Join(tmp, "config.yaml"), []byte(mainConfigYAML), 0644)

	// Write valid endpoint config
	_ = os.WriteFile(filepath.Join(tmp, "site1.yaml"), []byte(validEndpointConfigYAML), 0644)

	configs, err := LoadEnabledEndpointConfigs(tmp)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(configs) != 1 {
		t.Errorf("expected 1 config, got %d", len(configs))
	}
}

func TestLoadEnabledEndpointConfigs_DuplicateWithinFile(t *testing.T) {
	tmp := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmp, "config.yaml"), []byte(mainConfigYAML), 0644)
	_ = os.WriteFile(filepath.Join(tmp, "duplicate.yaml"), []byte(duplicateEndpointConfigYAML), 0644)

	_, err := LoadEnabledEndpointConfigs(tmp)
	if err == nil {
		t.Fatal("expected error for duplicate path in file, got nil")
	}
}

func TestLoadEnabledEndpointConfigs_DuplicateAcrossFiles(t *testing.T) {
	tmp := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmp, "config.yaml"), []byte(mainConfigYAML), 0644)

	// Two files with overlapping /path1
	endpointA := `
enable: true
endpoints:
  "/path1":
    strategy: round-robin
    urls:
      - url: "https://a.com"
`
	endpointB := `
enable: true
endpoints:
  "/path1":
    strategy: weighted
    urls:
      - url: "https://b.com"
        weight: 10
`

	_ = os.WriteFile(filepath.Join(tmp, "a.yaml"), []byte(endpointA), 0644)
	_ = os.WriteFile(filepath.Join(tmp, "b.yaml"), []byte(endpointB), 0644)

	_, err := LoadEnabledEndpointConfigs(tmp)
	if err == nil {
		t.Fatal("expected error for duplicate path across files, got nil")
	}
}
