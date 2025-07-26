package config

import (
	"os"
	"path/filepath"
	"testing"
)

const mainConfigYAML = `
port: 44562
https_cert_path: ""
https_key_path: ""
log:
  level: "info"
  output: "stdout"
  format: "text"
`

const validEndpointConfigYAML = `
enabled: true
endpoints:
  "/path1":
    strategy: round-robin
    urls:
      - url: "https://example.com/api1"
        socks5: "127.0.0.1:1080"
        username: "user1"
        password: "pass1"
      - url: "https://example.com/api2"
      - url: "https://example.com/api3"
    ban:
      - match: ["too many requests"]
        duration: 60
      - match: ["exceeded monthly allowance"]
        duration: 3600
  "/path2":
    strategy: weighted
    urls:
      - url: "https://example.com/api1"
        weight: 50
      - url: "https://example.com/api2"
        weight: 30
      - url: "https://example.com/api3"
        weight: 20
global_ban:
  - match: ["too many requests"]
    duration: 60
  - match: ["exceeded monthly allowance"]
    duration: 3600
`

const duplicateEndpointConfigYAML = `
enabled: true
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
	path := filepath.Join(tmp, "config.yaml")
	if err := os.WriteFile(path, []byte(mainConfigYAML), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadMainConfig(path)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.Port != 44562 {
		t.Errorf("expected port 44562, got: %d", cfg.Port)
	}
	if cfg.Log.Level != "info" || cfg.Log.Output != "stdout" || cfg.Log.Format != "text" {
		t.Errorf("unexpected log config: %+v", cfg.Log)
	}
}

func TestLoadEnabledEndpointsMap_Valid(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(mainConfigYAML), 0644)
	_ = os.WriteFile(filepath.Join(dir, "site1.yaml"), []byte(validEndpointConfigYAML), 0644)

	configs, err := LoadEnabledEndpointsMap(dir)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(configs) != 2 {
		t.Errorf("expected 2 endpoints, got %d", len(configs))
	}
}

func TestLoadEnabledEndpointsMap_DuplicateWithinFile(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "duplicate.yaml"), []byte(duplicateEndpointConfigYAML), 0644)

	_, err := LoadEnabledEndpointsMap(dir)
	if err == nil {
		t.Fatal("expected error due to duplicate endpoint in same file")
	}
}

func TestLoadEnabledEndpointsMap_DuplicateAcrossFiles(t *testing.T) {
	dir := t.TempDir()

	endpointA := `
enabled: true
endpoints:
  "/path1":
    strategy: round-robin
    urls:
      - url: "https://a.com"
    ban:
      - match: ["err"]
        duration: 1
global_ban:
  - match: ["global"]
    duration: 2
`
	endpointB := `
enabled: true
endpoints:
  "/path1":
    strategy: weighted
    urls:
      - url: "https://b.com"
        weight: 10
    ban:
      - match: ["err"]
        duration: 1
global_ban:
  - match: ["global"]
    duration: 2
`
	_ = os.WriteFile(filepath.Join(dir, "a.yaml"), []byte(endpointA), 0644)
	_ = os.WriteFile(filepath.Join(dir, "b.yaml"), []byte(endpointB), 0644)

	_, err := LoadEnabledEndpointsMap(dir)
	if err == nil {
		t.Fatal("expected error due to duplicate endpoint across files")
	}
}
