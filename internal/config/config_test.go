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

func TestFlattenBanRules(t *testing.T) {
	input := []BanRuleRaw{
		{
			Match:    []string{"word1", "word2"},
			Duration: 30,
		},
		{
			Match:    []string{"word3"},
			Duration: 60,
		},
	}

	expected := []BanRuleClean{
		{Match: "word1", Duration: 30},
		{Match: "word2", Duration: 30},
		{Match: "word3", Duration: 60},
	}

	result := flattenBanRules(input)

	if len(result) != len(expected) {
		t.Fatalf("expected %d rules, got %d", len(expected), len(result))
	}

	for i, rule := range result {
		if rule != expected[i] {
			t.Errorf("at index %d: expected %+v, got %+v", i, expected[i], rule)
		}
	}
}

func TestLoadEnabledEndpointsMap_BanRulesMergedCorrectly(t *testing.T) {
	dir := t.TempDir()

	// Local and global define "override" (local wins), and "global" (only global)
	endpointYAML := `
enabled: true
endpoints:
  "/merge":
    strategy: round-robin
    urls:
      - url: "https://example.com"
    ban:
      - match: ["override"]
        duration: 100
global_ban:
  - match: ["global"]
    duration: 10
  - match: ["override"]
    duration: 999
`

	if err := os.WriteFile(filepath.Join(dir, "merge.yaml"), []byte(endpointYAML), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	configs, err := LoadEnabledEndpointsMap(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg, ok := configs["/merge"]
	if !ok {
		t.Fatal("expected /merge endpoint to be loaded")
	}

	expected := map[string]int{
		"override": 100, // local wins
		"global":   10,  // from global
	}

	if len(cfg.BanRules) != len(expected) {
		t.Fatalf("expected %d ban rules, got %d", len(expected), len(cfg.BanRules))
	}

	for _, rule := range cfg.BanRules {
		expectedDuration, ok := expected[rule.Match]
		if !ok {
			t.Errorf("unexpected ban rule: %s", rule.Match)
		} else if rule.Duration != expectedDuration {
			t.Errorf("ban rule %s: expected duration %d, got %d", rule.Match, expectedDuration, rule.Duration)
		}
	}
}

func TestLoadEnabledEndpointsMap_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	badYAML := `enabled: true\nendpoints: {` // malformed
	_ = os.WriteFile(filepath.Join(dir, "bad.yaml"), []byte(badYAML), 0644)

	_, err := LoadEnabledEndpointsMap(dir)
	if err == nil {
		t.Fatal("expected error due to invalid YAML")
	}
}

func TestLoadMainConfig_Invalid(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"MissingPort", `
https_cert_path: ""
https_key_path: ""
log:
  level: "info"
  output: "stdout"
  format: "text"`},

		{"MissingLogLevel", `
port: 1234
https_cert_path: ""
https_key_path: ""
log:
  output: "stdout"
  format: "text"`},

		{"MissingLogOutput", `
port: 1234
https_cert_path: ""
https_key_path: ""
log:
  level: "info"
  format: "text"`},

		{"MissingLogFormat", `
port: 1234
https_cert_path: ""
https_key_path: ""
log:
  level: "info"
  output: "stdout"`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			path := filepath.Join(tmp, "config.yaml")
			if err := os.WriteFile(path, []byte(tc.content), 0644); err != nil {
				t.Fatal(err)
			}
			_, err := LoadMainConfig(path)
			if err == nil {
				t.Errorf("expected error for case %s, got nil", tc.name)
			}
		})
	}
}

func TestFindDuplicateEndpointWithinFile(t *testing.T) {
	yamlWithDup := `
endpoints:
  "/test1":
    strategy: round-robin
  "/test1":
    strategy: weighted
`
	dup := findDuplicateEndpointWithinFile([]byte(yamlWithDup))
	if dup != `"/test1":` {
		t.Errorf("expected duplicate '/test1', got: %s", dup)
	}
}
